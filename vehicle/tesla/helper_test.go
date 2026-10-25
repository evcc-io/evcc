package tesla

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestApiError(t *testing.T) {
	assert.Nil(t, apiError(nil))

	for _, msg := range []string{
		"408 Request Timeout",
		"408 (Request Timeout)",
		`408 Request Timeout: vehicle unavailable: {error: "vehicle unavailable:"}`,
		"vehicle unavailable: vehicle is offline or asleep",
	} {
		assert.ErrorIs(t, apiError(errors.New(msg)), api.ErrAsleep, msg)
	}

	for _, msg := range []string{
		"403 Forbidden: account disabled: EXCEEDED_LIMIT",
		"disconnected",
	} {
		assert.NotErrorIs(t, apiError(errors.New(msg)), api.ErrAsleep, msg)
	}
}
