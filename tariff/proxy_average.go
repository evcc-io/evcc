package tariff

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
)

// average wraps a tariff with averaging
type average struct {
	average time.Duration
	api.Tariff
}

// NewAverageProxy creates a proxy that tariff averaging
func NewAverageProxy(t api.Tariff) (api.Tariff, error) {
	return &average{
		average: time.Hour,
		Tariff:  t,
	}, nil
}

func (t *average) Rates() (api.Rates, error) {
	rates, err := t.Tariff.Rates()
	if len(rates) == 0 || err != nil {
		return rates, err
	}

	return averageSlots(rates, t.average), nil
}

// averageSlots averages 15-minute slots by period
func averageSlots(rates api.Rates, average time.Duration) api.Rates {
	if len(rates) == 0 {
		return nil
	}

	// accumulate sums and counts per period
	avgs := make(map[time.Time]*struct {
		sum float64
		cnt int
	})

	for _, r := range rates {
		ts := r.Start.Truncate(average)
		avg, ok := avgs[ts]
		if !ok {
			avg = new(struct {
				sum float64
				cnt int
			})
			avgs[ts] = avg
		}
		avg.sum += r.Value
		avg.cnt++
	}

	res := slices.Clone(rates)
	for i, r := range res {
		avg := avgs[r.Start.Truncate(average)]
		res[i].Value = avg.sum / float64(avg.cnt)
	}

	return res
}
