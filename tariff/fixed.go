package tariff

import (
	"fmt"
	"sort"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/fixed"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
)

type Fixed struct {
	clock   clock.Clock
	zones   fixed.Zones
	dynamic bool
}

var _ api.Tariff = (*Fixed)(nil)

func init() {
	registry.Add("fixed", NewFixedFromConfig)
}

func NewFixedFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Price float64
		Zones []struct {
			Price       float64
			Days, Hours string
		}
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &Fixed{
		clock:   clock.New(),
		dynamic: len(cc.Zones) > 1,
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

	// prepend catch-all zone
	t.zones = append([]fixed.Zone{
		{Price: cc.Price}, // full week is implicit
	}, t.zones...)

	return t, nil
}

// Rates implements the api.Tariff interface
func (t *Fixed) Rates() (api.Rates, error) {
	var res api.Rates

	start := now.With(t.clock.Now().Local()).BeginningOfDay()
	for i := 0; i < 7; i++ {
		dow := fixed.Day((int(start.Weekday()) + i) % 7)

		zones := t.zones.ForDay(dow)
		if len(zones) == 0 {
			return nil, fmt.Errorf("no zones for weekday %d", dow)
		}

		dayStart := start.AddDate(0, 0, i)
		markers := zones.TimeTableMarkers()

		for i, m := range markers {
			ts := dayStart.Add(time.Minute * time.Duration(m.Minutes()))

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
			end := dayStart.AddDate(0, 0, 1)
			if i+1 < len(markers) {
				end = dayStart.Add(time.Minute * time.Duration(markers[i+1].Minutes()))
			}

			rate := api.Rate{
				Price: zone.Price,
				Start: ts,
				End:   end,
			}

			res = append(res, rate)
		}
	}

	return res, nil
}

// Type implements the api.Tariff interface
func (t *Fixed) Type() api.TariffType {
	if t.dynamic {
		return api.TariffTypePriceDynamic
	}
	return api.TariffTypePriceStatic
}
