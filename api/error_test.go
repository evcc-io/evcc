package api

import (
	"fmt"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
)

// Backoff returns permanent errors unwrapped. These must still match the sentinel.
func TestPermanentSentinels(t *testing.T) {
	for _, tc := range []struct{ err, other error }{
		{ErrNotAvailable, ErrUnsupportedPlatform},
		{ErrUnsupportedPlatform, ErrNotAvailable},
		{ErrMissingCredentials, ErrMissingToken},
		{ErrMissingToken, ErrMissingCredentials},
	} {
		_, unwrapped := backoff.RetryWithData(func() (int, error) {
			return 0, tc.err
		}, &backoff.StopBackOff{})

		assert.ErrorIs(t, unwrapped, tc.err)
		assert.ErrorIs(t, fmt.Errorf("wrapped: %w", unwrapped), tc.err)
		assert.NotErrorIs(t, unwrapped, tc.other)
	}
}
