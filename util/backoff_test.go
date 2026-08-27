package util

import (
	"testing"
	"time"

	"github.com/andig/backoff"
	"github.com/stretchr/testify/assert"
)

// BackOff must keep the library defaults for options that are not given
func TestBackOffDefaults(t *testing.T) {
	bo := BackOff()

	assert.Equal(t, backoff.DefaultInitialInterval, bo.InitialInterval)
	assert.Equal(t, backoff.DefaultRandomizationFactor, bo.RandomizationFactor)
	assert.Equal(t, backoff.DefaultMultiplier, bo.Multiplier)
	assert.Equal(t, backoff.DefaultMaxInterval, bo.MaxInterval)
}

func TestBackOffOptions(t *testing.T) {
	bo := BackOff(WithInitialInterval(time.Second), WithMaxInterval(time.Minute), WithMultiplier(3))

	assert.Equal(t, time.Second, bo.InitialInterval)
	assert.Equal(t, time.Minute, bo.MaxInterval)
	assert.Equal(t, 3.0, bo.Multiplier)
	assert.Equal(t, backoff.DefaultRandomizationFactor, bo.RandomizationFactor)
}

// Callers sleep on the raw NextBackOff value, so it must never turn negative
func TestBackOffNeverStops(t *testing.T) {
	bo := BackOff(WithInitialInterval(time.Millisecond), WithMaxInterval(10*time.Millisecond))

	for range 100 {
		assert.Positive(t, bo.NextBackOff())
	}
}
