package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

// TestHomeAssistantBatteryModes covers which modes a partial mode config announces
func TestHomeAssistantBatteryModes(t *testing.T) {
	conf := func(hold, charge string) map[string]any {
		return map[string]any{
			"uri":        "http://localhost",
			"token":      "foo",
			"power":      "sensor.power",
			"soc":        "sensor.soc",
			"modeNormal": "script.normal",
			"modeHold":   hold,
			"modeCharge": charge,
		}
	}

	m, err := NewHomeAssistantFromConfig(conf("script.hold", "script.charge"))
	require.NoError(t, err)

	ctrl, ok := api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge}, ctrl.BatteryModes())

	// a mode without entity is not announced and rejected by the setter
	m, err = NewHomeAssistantFromConfig(conf("script.hold", ""))
	require.NoError(t, err)

	ctrl, ok = api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold}, ctrl.BatteryModes())
	require.Error(t, ctrl.SetBatteryMode(api.BatteryCharge))

	// a mode entity must be a script
	_, err = NewHomeAssistantFromConfig(conf("switch.hold", ""))
	require.Error(t, err)
}
