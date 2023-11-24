package tariff

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
)

func newBackoff() backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.MaxElapsedTime = time.Minute
	return bo
}

type LeveledRate struct {
	api.Rate
	Cheap bool `json:"cheap"`
}

func LevelRates(rates api.Rates, percent float64) []LeveledRate {
	res := make([]LeveledRate, len(rates))

	var (
		totalDuration time.Duration
		weighedCost   float64
	)

	for _, r := range rates {
		d := r.End.Sub(r.Start)
		totalDuration += d
		weighedCost += r.Price * float64(d)
	}

	if totalDuration > 0 {
		weighedCost /= float64(totalDuration)
	}

	cheap := weighedCost * percent
	for i, r := range rates {
		res[i].Rate = r
		if r.Price <= cheap {
			res[i].Cheap = true
		}
	}

	return res
}
