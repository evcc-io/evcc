package fixed

import "sort"

type ZoneConfig []struct {
	Price       float64
	Days, Hours string
}

func (zz ZoneConfig) Parse(base float64) (Zones, error) {
	var res Zones

	for _, z := range zz {
		days, err := ParseDays(z.Days)
		if err != nil {
			return nil, err
		}

		hours, err := ParseTimeRanges(z.Hours)
		if err != nil && z.Hours != "" {
			return nil, err
		}

		if len(hours) == 0 {
			res = append(res, Zone{
				Price: z.Price,
				Days:  days,
			})
			continue
		}

		for _, h := range hours {
			res = append(res, Zone{
				Price: z.Price,
				Days:  days,
				Hours: h,
			})
		}
	}

	sort.Sort(res)

	// prepend catch-all zone
	res = append([]Zone{
		{Price: base}, // full week is implicit
	}, res...)

	return res, nil
}
