package circuit

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// bo returns an exponential backoff for reading meter power quickly
func bo() *backoff.ExponentialBackOff {
	return backoff.NewExponentialBackOff(backoff.WithInitialInterval(20*time.Millisecond), backoff.WithMaxElapsedTime(time.Second))
}
