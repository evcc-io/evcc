package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestHomeAssistantMilliamps(t *testing.T) {
	for _, milliamps := range []bool{false, true} {
		c, err := NewHomeAssistantFromConfig(map[string]any{
			"uri":        "http://localhost:8123",
			"status":     "sensor.status",
			"enabled":    "sensor.enabled",
			"enable":     "switch.enable",
			"maxcurrent": "number.maxcurrent",
			"milliamps":  milliamps,
		})
		require.NoError(t, err)

		_, ok := api.Cap[api.ChargerEx](c)
		require.Equal(t, milliamps, ok, "milliamps=%v", milliamps)
	}
}
