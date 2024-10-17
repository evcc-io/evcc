package fixed

import (
	"slices"
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
