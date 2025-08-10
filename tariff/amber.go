package tariff

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/amber"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type Amber struct {
	*embed
	*request.Helper
	log     *util.Logger
	uri     string
	channel string
	data    *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Amber)(nil)

func init() {
	registry.Add("amber", NewAmberFromConfig)
}

func NewAmberFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed   `mapstructure:",squash"`
		Token   string
		SiteID  string
		Channel string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Token == "" {
		return nil, api.ErrMissingToken
	}

	if cc.SiteID == "" {
		return nil, errors.New("missing siteid")
	}

	if cc.Channel == "" {
		return nil, errors.New("missing channel")
	}

	log := util.NewLogger("amber").Redact(cc.Token)

	t := &Amber{
		embed:   &cc.embed,
		log:     log,
		Helper:  request.NewHelper(log),
		uri:     fmt.Sprintf(amber.URI, strings.ToUpper(cc.SiteID)),
		channel: strings.ToLower(cc.Channel),
		data:    util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = &transport.Decorator{
		Base: t.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + cc.Token,
		}),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Amber) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Minute); ; <-tick {
		var res []amber.PriceInfo

		if err := backoff.Retry(func() error {
			return backoffPermanentError(t.GetJSON(t.uri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		// Group by hour and average intervals within each hour
		hourlyData := make(map[time.Time]*struct {
			totalValue    float64
			totalDuration time.Duration
			start         time.Time
			currentValue  *float64 // Override with current interval if present (for accurate charging session costs)
		})

		for _, r := range res {
			if t.channel == strings.ToLower(r.ChannelType) {
				startTime, _ := time.Parse("2006-01-02T15:04:05Z", r.StartTime)
				endTime, _ := time.Parse("2006-01-02T15:04:05Z", r.EndTime)

				value := r.PerKwh / 1e2
				if r.AdvancedPrice != nil {
					value = r.AdvancedPrice.Predicted / 1e2
				}

				// Invert feed-in prices to match evcc expectations (positive = paid for exports)
				if t.channel == "feedin" {
					value = -value
				}

				localStart := startTime.Local()
				localEnd := endTime.Local()
				hourStart := localStart.Truncate(time.Hour) // Preserve date+hour
				duration := localEnd.Sub(localStart)

				// Initialize hour entry if needed
				if hourlyData[hourStart] == nil {
					hourlyData[hourStart] = &struct {
						totalValue    float64
						totalDuration time.Duration
						start         time.Time
						currentValue  *float64
					}{start: hourStart}
				}

				hr := hourlyData[hourStart]

				// If this is the current interval, use its value directly for this hour
				if r.Type == "CurrentInterval" {
					hr.currentValue = &value
				} else {
					// Add to weighted average for forecast intervals
					hr.totalValue += value * duration.Seconds()
					hr.totalDuration += duration
				}
			}
		}

		// Convert to final hourly rates
		data := make(api.Rates, 0, len(hourlyData))
		for _, hr := range hourlyData {
			var finalValue float64
			if hr.currentValue != nil {
				// Use current interval value if available
				finalValue = *hr.currentValue
			} else if hr.totalDuration > 0 {
				// Otherwise use weighted average of forecast intervals
				finalValue = hr.totalValue / hr.totalDuration.Seconds()
			}

			data = append(data, api.Rate{
				Start: hr.start,
				End:   hr.start.Add(time.Hour),
				Value: finalValue,
			})
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Amber) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

func (t *Amber) Unit() string {
	return "AUD"
}

// Type returns the tariff type
func (t *Amber) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
