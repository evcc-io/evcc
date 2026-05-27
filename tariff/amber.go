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

func NewAmberFromConfig(other map[string]any) (api.Tariff, error) {
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

		// Create and sort time-ordered list of all Amber intervals
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

		// Sort intervals by start time to ensure correct processing
		slices.SortFunc(intervals, func(a, b struct {
			start, end time.Time
			value      float64
			isCurrent  bool
		}) int {
			return a.start.Compare(b.start)
		})

		data := t.buildSlotRates(intervals)

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// buildSlotRates converts Amber intervals into 15-minute slots using bucket sharding
// to avoid O(slots × intervals) complexity and only create slots with actual data
func (t *Amber) buildSlotRates(intervals []struct {
	start, end time.Time
	value      float64
	isCurrent  bool
}) api.Rates {
	// Build slot buckets using sharding approach
	type bucket struct {
		totalSecs   float64
		weightedSum float64
		current     *float64
	}
	buckets := make(map[time.Time]*bucket)

	for _, iv := range intervals {
		// Truncate start to slot boundary
		slot := iv.start.Truncate(SlotDuration)
		end := iv.end

		for slot.Before(end) {
			next := slot.Add(SlotDuration)

			// Compute overlap [max(slot, iv.start), min(next, iv.end))
			overlapStart := slot
			if iv.start.After(slot) {
				overlapStart = iv.start
			}

			overlapEnd := next
			if iv.end.Before(next) {
				overlapEnd = iv.end
			}

			overlapSecs := overlapEnd.Sub(overlapStart).Seconds()

			b, ok := buckets[slot]
			if !ok {
				b = &bucket{}
				buckets[slot] = b
			}

			if iv.isCurrent {
				// Current interval overrides the entire slot
				b.current = &iv.value
			} else {
				// Add to weighted average
				b.weightedSum += iv.value * overlapSecs
				b.totalSecs += overlapSecs
			}

			slot = next
		}
	}

	// Convert buckets to sorted rates, skipping empty slots
	var data api.Rates
	for start, b := range buckets {
		var finalValue float64
		hasValue := false

		if b.current != nil {
			finalValue = *b.current
			hasValue = true
		} else if b.totalSecs > 0 {
			finalValue = b.weightedSum / b.totalSecs
			hasValue = true
		}

		// Only add slots with actual data
		if hasValue {
			data = append(data, api.Rate{
				Start: start,
				End:   start.Add(SlotDuration),
				Value: finalValue,
			})
		}
	}

	// Sort by start time
	slices.SortFunc(data, func(a, b api.Rate) int {
		return a.Start.Compare(b.Start)
	})

	return data
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
