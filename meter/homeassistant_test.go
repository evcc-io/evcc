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

	// holdcharge and discharge are announced when configured
	c := conf("", "")
	c["modeHoldCharge"] = "script.holdcharge"
	c["modeDischarge"] = "script.discharge"
	m, err = NewHomeAssistantFromConfig(c)
	require.NoError(t, err)

	ctrl, ok = api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHoldCharge, api.BatteryDischarge}, ctrl.BatteryModes())

	// modeNormal is required with any other mode
	c = conf("", "")
	c["modeNormal"] = ""
	c["modeDischarge"] = "script.discharge"
	_, err = NewHomeAssistantFromConfig(c)
	require.Error(t, err)

	// modeNormal alone is rejected
	_, err = NewHomeAssistantFromConfig(conf("", ""))
	require.Error(t, err)
}
