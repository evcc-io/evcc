package fixed

import (
	"fmt"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
)

type Zone struct {
	Price float64
	Days  []Day
	Hours TimeRange
}

type Zones []Zone

// implement sort.Interface
func (r Zones) Len() int {
	return len(r)
}

func (r Zones) Less(i, j int) bool {
	if r[i].Hours.From.Minutes() == r[j].Hours.From.Minutes() {
		return r[i].Hours.To.Minutes() > r[j].Hours.To.Minutes()
	}
	return r[i].Hours.From.Minutes() < r[j].Hours.From.Minutes()
}

func (r Zones) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// ForDay returns the zones for given day in ascending order
func (r Zones) ForDay(day Day) Zones {
	var zones Zones
	for _, z := range r {
		if slices.Contains(z.Days, day) || len(z.Days) == 0 {
			zones = append(zones, z)
		}
	}

	return zones
}

// TimeTableMarkers returns list of zone start/end markers
func (r Zones) TimeTableMarkers() []HourMin {
	res := []HourMin{{Hour: 0, Min: 0}}

	for _, z := range r {
		if !z.Hours.From.IsNil() {
			res = append(res, z.Hours.From)
		}
		if !z.Hours.To.IsNil() {
			res = append(res, z.Hours.To)
		}
	}

HOURS:
	// 1hr intervals
	for hour := 0; hour < 24; hour++ {
		for _, m := range res {
			if m.Hour == hour && m.Min == 0 {
				continue HOURS
			}
		}

		// hour is missing
		for i, m := range res {
			if m.Hour >= hour {
				res = slices.Insert(res, i, HourMin{Hour: hour, Min: 0})
				continue HOURS
			}
		}

		res = append(res, HourMin{Hour: hour, Min: 0})
	}

	return res
}

// Rates implements the api.Tariff interface
func (r Zones) Rates(tNow time.Time) (api.Rates, error) {
	var res api.Rates

	start := now.With(tNow.Local()).BeginningOfDay()
	for i := 0; i < 7; i++ {
		dow := Day((int(start.Weekday()) + i) % 7)

		zones := r.ForDay(dow)
		if len(zones) == 0 {
			return nil, fmt.Errorf("no zones for weekday %d", dow)
		}

		dayStart := start.AddDate(0, 0, i)
		markers := zones.TimeTableMarkers()

		for i, m := range markers {
			ts := dayStart.Add(time.Minute * time.Duration(m.Minutes()))

			var zone *Zone
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
