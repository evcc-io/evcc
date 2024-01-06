package tariff

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util/request"
)

func newBackoff() backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.MaxElapsedTime = time.Minute
	return bo
}

// backoffPermanentError returns a permanent error in case of HTTP 400
func backoffPermanentError(err error) error {
	if se, ok := err.(request.StatusError); ok {
		if code := se.StatusCode(); code >= 400 && code < 500 {
			return backoff.Permanent(se)
		}
	}
	return err
}
