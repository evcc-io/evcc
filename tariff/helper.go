package tariff

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jinzhu/now"
)

// Name returns the tariff type name
func Name(conf config.Typed) string {
	if conf.Other != nil && conf.Other["tariff"] != nil {
		return conf.Other["tariff"].(string)
	}
	return conf.Type
}

func bo() backoff.BackOff {
	return backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Second),
		backoff.WithMaxElapsedTime(time.Minute),
	)
}

// backoffPermanentError returns a permanent error in case of non-recoverable http errors
func backoffPermanentError(err error) error {
	return request.BackoffDefaultHttpStatusCodesPermanently()(err)
}

// mergeRates blends new and existing rates, keeping existing rates after current hour
func mergeRates(data *util.Monitor[api.Rates], new api.Rates) {
	mergeRatesAfter(data, new, now.With(time.Now()).BeginningOfHour())
}

// mergeRatesAfter blends new and existing rates, keeping existing rates after timestamp
func mergeRatesAfter(data *util.Monitor[api.Rates], new api.Rates, now time.Time) {
	new.Sort()

	var newStart time.Time
	if len(new) > 0 {
		newStart = new[0].Start
	}

	data.SetFunc(func(old api.Rates) api.Rates {
		var between api.Rates
		for _, r := range old {
			if (r.Start.Before(newStart) && !r.Start.Before(now) || newStart.IsZero()) && r.End.After(now) {
				between = append(between, r)
			}
		}

		return append(between, new...)
	})
}

// beginningOfDay returns the beginning of the current day
func beginningOfDay() time.Time {
	return now.With(time.Now()).BeginningOfDay()
}
