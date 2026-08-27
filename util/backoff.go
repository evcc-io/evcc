package util

import (
	"time"

	"github.com/andig/backoff"
)

// BackOffOption configures an exponential backoff policy
type BackOffOption func(*backoff.ExponentialBackOff)

// WithInitialInterval sets the first retry interval
func WithInitialInterval(d time.Duration) BackOffOption {
	return func(bo *backoff.ExponentialBackOff) { bo.InitialInterval = d }
}

// WithMaxInterval caps the retry interval
func WithMaxInterval(d time.Duration) BackOffOption {
	return func(bo *backoff.ExponentialBackOff) { bo.MaxInterval = d }
}

// WithMultiplier sets the factor the retry interval grows by
func WithMultiplier(f float64) BackOffOption {
	return func(bo *backoff.ExponentialBackOff) { bo.Multiplier = f }
}

// BackOff returns an exponential backoff policy, applying opts to the defaults
func BackOff(opts ...BackOffOption) *backoff.ExponentialBackOff {
	bo := backoff.NewExponentialBackOff()
	for _, opt := range opts {
		opt(bo)
	}
	return bo
}
