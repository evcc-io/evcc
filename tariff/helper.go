package tariff

import (
	"net/http"
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
		if se.HasStatus(http.StatusBadRequest) {
			return backoff.Permanent(se)
		}
	}
	return err
}
