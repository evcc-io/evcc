package tariff

import (
	"errors"
	"strings"
	"sync"
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
	if se, ok := errors.AsType[*request.StatusError](err); ok {
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

// reportError reports the first error to done via once and returns true when
// this is the startup failure - i.e. run's first update failed and runOrError
// is about to discard the tariff. The caller must then return so the goroutine
// exits instead of polling the API forever in the background.
//
// Once a tariff has started successfully, once has already fired (with close),
// so reportError is a no-op returning false and the caller keeps retrying
// transient errors as before.
func reportError(once *sync.Once, done chan<- error, err error) (startupFailed bool) {
	once.Do(func() {
		startupFailed = true
		done <- err
	})
	return startupFailed
}
