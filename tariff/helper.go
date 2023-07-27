package tariff

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

func newBackoff() backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 5 * time.Second
	bo.MaxElapsedTime = time.Minute
	return bo
}
