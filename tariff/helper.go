package tariff

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

func newBackoff() backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.MaxInterval = time.Minute
	bo.MaxElapsedTime = time.Minute
	return bo
}
