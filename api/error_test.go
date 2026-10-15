package api

import (
	"fmt"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
)

// permanentSentinels are all errors created by permanent()
var permanentSentinels = []error{
	ErrNotAvailable,
	ErrUnsupportedPlatform,
	ErrSponsorRequired,
	ErrMissingCredentials,
	ErrMissingToken,
}

// Backoff returns permanent errors unwrapped. These must still match the sentinel.
func TestPermanentSentinels(t *testing.T) {
	for _, err := range permanentSentinels {
		_, unwrapped := backoff.RetryWithData(func() (int, error) {
			return 0, err
		}, &backoff.StopBackOff{})

		assert.ErrorIs(t, unwrapped, err)
		assert.ErrorIs(t, fmt.Errorf("wrapped: %w", unwrapped), err)
		assert.ErrorIs(t, fmt.Errorf("wrapped: %w", err), err)
	}
}

// Permanent sentinels must not match each other, whether returned directly or
// unwrapped by backoff.
func TestPermanentSentinelIdentity(t *testing.T) {
	for _, err := range permanentSentinels {
		_, unwrapped := backoff.RetryWithData(func() (int, error) {
			return 0, err
		}, &backoff.StopBackOff{})

		for _, other := range permanentSentinels {
			if other == err {
				continue
			}

			assert.NotErrorIs(t, err, other)
			assert.NotErrorIs(t, unwrapped, other)
			assert.NotErrorIs(t, fmt.Errorf("wrapped: %w", err), other)
		}
	}

	// login required is permanent, too
	assert.NotErrorIs(t, LoginRequiredError("foo"), ErrNotAvailable)
}
