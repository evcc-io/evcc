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

	return runOrError(t)
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

		// Create a time-ordered list of all Amber intervals
		var intervals []struct {
			start, end time.Time
			value      float64
			isCurrent  bool
		}

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

				intervals = append(intervals, struct {
					start, end time.Time
					value      float64
					isCurrent  bool
				}{
					start:     startTime.Local(),
					end:       endTime.Local(),
					value:     value,
					isCurrent: r.Type == "CurrentInterval",
				})
			}
		}

		if len(intervals) == 0 {
			mergeRates(t.data, nil)
			once.Do(func() { close(done) })
			continue
		}

		// Find time range and create 15-minute slots
		minTime := intervals[0].start.Truncate(SlotDuration)
		maxTime := intervals[len(intervals)-1].end

		var data api.Rates
		for slotStart := minTime; slotStart.Before(maxTime); slotStart = slotStart.Add(SlotDuration) {
			slotEnd := slotStart.Add(SlotDuration)

			var totalValue, totalDuration float64
			var currentPrice *float64

			// Find all intervals that overlap with this 15-minute slot
			for _, interval := range intervals {
				if interval.end.After(slotStart) && interval.start.Before(slotEnd) {
					// Calculate overlap duration
					overlapStart := slotStart
					if interval.start.After(slotStart) {
						overlapStart = interval.start
					}

					overlapEnd := slotEnd
					if interval.end.Before(slotEnd) {
						overlapEnd = interval.end
					}

					overlapSecs := overlapEnd.Sub(overlapStart).Seconds()

					if interval.isCurrent {
						// Current interval overrides the entire slot
						currentPrice = &interval.value
					} else {
						// Add to weighted average
						totalValue += interval.value * overlapSecs
						totalDuration += overlapSecs
					}
				}
			}

			// Determine final value for this slot
			var finalValue float64
			if currentPrice != nil {
				finalValue = *currentPrice
			} else if totalDuration > 0 {
				finalValue = totalValue / totalDuration
			}

			data = append(data, api.Rate{
				Start: slotStart,
				End:   slotEnd,
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
