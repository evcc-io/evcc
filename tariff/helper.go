package tariff

import (
	"errors"
	"strings"
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

// backoffPermanentError returns a permanent error in case of HTTP 400
func backoffPermanentError(err error) error {
	if se := new(request.StatusError); errors.As(err, &se) {
		if code := se.StatusCode(); code >= 400 && code <= 599 {
			return backoff.Permanent(se)
		}
	}
	if err != nil && strings.HasPrefix(err.Error(), "jq: query failed") {
		return backoff.Permanent(err)
	}
	return err
}

// mergeRates blends new and existing rates, keeping existing rates after current hour
func mergeRates(data *util.Monitor[api.Rates], new api.Rates) {
	mergeRatesAfter(data, new, time.Now().Truncate(SlotDuration))
}

// mergeRatesAfter blends new and existing rates, keeping existing rates after timestamp
func mergeRatesAfter(data *util.Monitor[api.Rates], new api.Rates, now time.Time) {
	new.Sort()

	var newStart time.Time
	if len(new) > 0 {
		new.Sort()
		newStart = new[0].Start
	}

	data.SetFunc(func(old api.Rates) api.Rates {
		var between api.Rates
		for _, r := range old {
			if (newStart.IsZero() || !r.End.After(newStart)) && r.End.After(now) {
				between = append(between, r)
			}
		}

		return append(between, new...)
	})
}

// beginningOfDay returns the beginning of the current day
func beginningOfDay() time.Time {
	return now.BeginningOfDay()
}

type runnable[T any] interface {
	*T
	run(done chan error)
}

// https://groups.google.com/g/golang-nuts/c/1cl9v_hPYHk
// runOrError invokes t.run(chan error) and waits for the channel to return
func runOrError[T any, I runnable[T]](t I) (*T, error) {
	done := make(chan error)
	go t.run(done)

	if err := <-done; err != nil {
		return nil, err
	}

	return t, nil
}
