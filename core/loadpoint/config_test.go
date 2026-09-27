package loadpoint

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// expectCommonApplySetters stubs the DynamicConfig.Apply setters that are unrelated
// to default mode/always charge so tests can focus on their interaction.
func expectCommonApplySetters(m *MockAPI) {
	m.EXPECT().SetTitle(gomock.Any()).AnyTimes()
	m.EXPECT().SetPriority(gomock.Any()).AnyTimes()
	m.EXPECT().SetSmartCostLimit(gomock.Any()).AnyTimes()
	m.EXPECT().SetSmartFeedInPriorityLimit(gomock.Any()).AnyTimes()
	m.EXPECT().SetSolarShare(gomock.Any()).AnyTimes()
	m.EXPECT().SetThresholds(gomock.Any()).AnyTimes()
	m.EXPECT().SetPlanEnergy(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	m.EXPECT().SetPlanStrategy(gomock.Any()).Return(nil).AnyTimes()
	m.EXPECT().SetBatteryBoostLimit(gomock.Any()).AnyTimes()
	m.EXPECT().SetLimitEnergy(gomock.Any()).AnyTimes()
	m.EXPECT().SetLimitSoc(gomock.Any()).AnyTimes()
	m.EXPECT().SetMinSoc(gomock.Any()).AnyTimes()
	m.EXPECT().SetSocConfig(gomock.Any()).AnyTimes()
	m.EXPECT().SetUI(gomock.Any()).AnyTimes()
	m.EXPECT().SetPhasesConfigured(gomock.Any()).Return(nil).AnyTimes()
}

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

// TestSplitConfigAlwaysCharge guards that alwaysCharge is treated as dynamic
// config. It is persisted per loadpoint (and, for database-configured loadpoints,
// written back into the config blob by the pv/minpv self-heal migration), so it
// must not reach the strict static decode in NewLoadpointFromConfig.
func TestSplitConfigAlwaysCharge(t *testing.T) {
	payload := map[string]any{
		"charger":      "wallbox",
		"alwaysCharge": "on",
	}

	dynamic, other, err := SplitConfig(payload)
	require.NoError(t, err)

	assert.Equal(t, "on", dynamic.AlwaysCharge)
	assert.NotContains(t, other, "alwaysCharge")
}

// TestApplySeedsAlwaysChargeFromLegacyDefaultMode guards that database-configured
// loadpoints seed always charge from a legacy minpv default exactly like
// NewLoadpointFromConfig does for yaml loadpoints, since SetDefaultMode itself must
// not touch always charge on later, explicit calls.
func TestApplySeedsAlwaysChargeFromLegacyDefaultMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockAPI(ctrl)
	expectCommonApplySetters(m)

	// first boot: alwaysCharge was never persisted, the legacy default seeds it once
	m.EXPECT().SetAlwaysCharge(api.AlwaysChargeOn).Return(nil)
	m.EXPECT().SetDefaultMode(api.ModeSmart)

	require.NoError(t, DynamicConfig{DefaultMode: "minpv"}.Apply(m))
}

func TestApplyDoesNotReseedAlwaysCharge(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockAPI(ctrl)
	expectCommonApplySetters(m)

	// later boots: alwaysCharge is already persisted, the legacy default must not override it
	m.EXPECT().SetAlwaysCharge(api.AlwaysChargeOff).Return(nil)
	m.EXPECT().SetDefaultMode(api.ModeSmart)

	require.NoError(t, DynamicConfig{DefaultMode: "minpv", AlwaysCharge: "off"}.Apply(m))
}
