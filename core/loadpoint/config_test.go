package loadpoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitConfigUI(t *testing.T) {
	payload := map[string]any{
		"title": "Water Heater",
		"ui": map[string]any{
			"minTemp": 20.0,
			"maxTemp": 45.0,
		},
	}

	dynamic, other, err := SplitConfig(payload)
	require.NoError(t, err)

	assert.Equal(t, 20.0, dynamic.UI.MinTemp)
	assert.Equal(t, 45.0, dynamic.UI.MaxTemp)
	assert.NotContains(t, other, "ui")
}
