package api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"
)

// TestPermanentSentinelSurvivesBackoff ensures permanent sentinels stay
// identifiable after backoff.Retry, which returns PermanentError.Err.
func TestPermanentSentinelSurvivesBackoff(t *testing.T) {
	for _, sentinel := range []error{ErrNotAvailable, ErrUnsupportedPlatform, ErrMissingCredentials, ErrMissingToken} {
		t.Run(sentinel.Error(), func(t *testing.T) {
			var calls int

			_, err := backoff.RetryWithData(func() (float64, error) {
				calls++
				return 0, sentinel
			}, backoff.NewExponentialBackOff())

			require.Equal(t, 1, calls, "must not retry")
			require.ErrorIs(t, err, sentinel)
		})
	}
}

// TestPermanentSentinelWrapped ensures a wrapped sentinel is still permanent.
func TestPermanentSentinelWrapped(t *testing.T) {
	var calls int

	_, err := backoff.RetryWithData(func() (float64, error) {
		calls++
		return 0, fmt.Errorf("add[2]: %w", ErrNotAvailable)
	}, backoff.NewExponentialBackOff())

	require.Equal(t, 1, calls, "must not retry")
	require.ErrorIs(t, err, ErrNotAvailable)
}

// TestPermanentSentinelDistinct ensures sentinels don't match each other.
func TestPermanentSentinelDistinct(t *testing.T) {
	require.False(t, errors.Is(ErrNotAvailable, ErrMissingToken))
}
