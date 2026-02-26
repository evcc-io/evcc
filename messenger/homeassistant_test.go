package messenger

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHAMessengerMissingURI(t *testing.T) {
	_, err := NewHAMessengerFromConfig(map[string]any{})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "uri"))
}

func TestHAMessengerInvalidNotify(t *testing.T) {
	for _, notify := range []string{"notify", "notify.", ".foo"} {
		_, err := NewHAMessengerFromConfig(map[string]any{
			"uri":    "ws://localhost:8123",
			"notify": notify,
		})
		require.Error(t, err, "expected error for notify=%q", notify)
		assert.Contains(t, err.Error(), "domain.service", "notify=%q", notify)
	}
}
