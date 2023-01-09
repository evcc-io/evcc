package tariff

import (
	"fmt"
	"sort"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/fixed"
	"github.com/evcc-io/evcc/util"
	"github.com/golang-module/carbon"
)

type Fixed struct {
	unit  string
	clock clock.Clock
	zones fixed.Zones
}

var _ api.Tariff = (*Fixed)(nil)

func init() {
	registry.Add("fixed", NewFixedFromConfig)
}

func NewFixedFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Currency string
		Price    float64
		Zones    []struct {
			Price       float64
			Days, Hours string
		}
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Currency == "" {
		cc.Currency = "EUR"
	}

	t := &Fixed{
		unit:  cc.Currency,
		clock: clock.New(),
		zones: []fixed.Zone{
			{Price: cc.Price}, // full week is implicit
		},
	}

	for _, z := range cc.Zones {
		days, err := fixed.ParseDays(z.Days)
		if err != nil {
			return nil, err
		}

		hours, err := fixed.ParseTimeRanges(z.Hours)
		if err != nil && z.Hours != "" {
			return nil, err
		}

		if len(hours) == 0 {
			t.zones = append(t.zones, fixed.Zone{
				Price: z.Price,
				Days:  days,
			})
			continue
		}

		for _, h := range hours {
			t.zones = append(t.zones, fixed.Zone{
				Price: z.Price,
				Days:  days,
				Hours: h,
			})
		}
	}

	sort.Sort(t.zones)

	return t, nil
}

// Unit implements the api.Tariff interface
func (t *Fixed) Unit() string {
	return t.unit
}

// Rates implements the api.Tariff interface
func (t *Fixed) Rates() (api.Rates, error) {
	var res api.Rates

	start := carbon.Time2Carbon(t.clock.Now().Local()).StartOfDay()
	for i := 0; i < 7; i++ {
		dow := fixed.Day((start.DayOfWeek() + i) % 7)

		zones := t.zones.ForDay(dow)
		if len(zones) == 0 {
			return nil, fmt.Errorf("no zones for weekday %d", dow)
		}

		dayStart := start.AddDays(i)
		markers := zones.TimeTableMarkers()

		for i, m := range markers {
			ts := dayStart.AddMinutes(m.Minutes())

			var zone *fixed.Zone
			for j := len(zones) - 1; j >= 0; j-- {
				if zones[j].Hours.Contains(m) {
					zone = &zones[j]
					break
				}
			}

			if zone == nil {
				return nil, fmt.Errorf("could not find zone for %02d:%02d", m.Hour, m.Min)
			}

			// end rate at end of day or next marker
			end := dayStart.AddDay()
			if i+1 < len(markers) {
				end = dayStart.AddMinutes(markers[i+1].Minutes())
			}

			rate := api.Rate{
				Price: zone.Price,
				Start: ts.Carbon2Time(),
				End:   end.Carbon2Time(),
			}

			res = append(res, rate)
		}
	}

	return res, nil
}
