package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestSettingsRoundtrip asserts that settings written through the API are
// restored into a fresh loadpoint sharing the same store.
func TestSettingsRoundtrip(t *testing.T) {
	store := settings.NewMemorySettings()
	uiChan, pushChan, lpChan := createChannels(t)

	planTime := time.Now().Add(4 * time.Hour).Round(time.Second)
	smartCost := 0.25
	feedInPriority := 0.15
	estimate := true

	soc := loadpoint.SocConfig{
		Poll:     loadpoint.PollConfig{Mode: loadpoint.PollAlways, Interval: 42 * time.Minute},
		Estimate: &estimate,
	}
	thresholds := loadpoint.ThresholdsConfig{
		Enable:  loadpoint.ThresholdConfig{Delay: 2 * time.Minute, Threshold: -100},
		Disable: loadpoint.ThresholdConfig{Delay: 4 * time.Minute, Threshold: 200},
	}
	strategy := api.PlanStrategy{Continuous: true, Precondition: 30 * time.Minute}

	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	charger.EXPECT().Enabled().Return(false, nil).AnyTimes()

	// write
	lp := NewLoadpoint(util.NewLogger("foo"), store)
	lp.charger = charger
	lp.Prepare(new(Site), uiChan, pushChan, lpChan)

	lp.SetMode(api.ModeSmart)
	lp.SetPriority(3)
	require.NoError(t, lp.SetMinCurrent(8))
	require.NoError(t, lp.SetMaxCurrent(12))
	lp.SetLimitSoc(75)
	lp.SetLimitEnergy(11)
	lp.SetMinSoc(20)
	lp.SetSmartCostLimit(&smartCost)
	lp.SetSmartFeedInPriorityLimit(&feedInPriority)
	lp.SetBatteryBoostLimit(90)
	lp.SetThresholds(thresholds)
	lp.SetSocConfig(soc)
	require.NoError(t, lp.SetPlanEnergy(planTime, 7))
	require.NoError(t, lp.SetPlanStrategy(strategy))

	// read back into a fresh loadpoint
	restored := NewLoadpoint(util.NewLogger("bar"), store)
	restored.charger = charger
	restored.Prepare(new(Site), uiChan, pushChan, lpChan)

	assert.Equal(t, api.ModeSmart, restored.GetMode())
	assert.Equal(t, 3, restored.GetPriority())
	assert.Equal(t, 8.0, restored.GetMinCurrent())
	assert.Equal(t, 12.0, restored.GetMaxCurrent())
	assert.Equal(t, 75, restored.GetLimitSoc())
	assert.Equal(t, 11.0, restored.GetLimitEnergy())
	assert.Equal(t, 20, restored.GetMinSoc())
	assert.Equal(t, &smartCost, restored.GetSmartCostLimit())
	assert.Equal(t, &feedInPriority, restored.GetSmartFeedInPriorityLimit())
	assert.Equal(t, 90, restored.GetBatteryBoostLimit())
	assert.Equal(t, thresholds, restored.GetThresholds())
	assert.Equal(t, soc, restored.GetSocConfig())
	assert.Equal(t, strategy, restored.GetPlanStrategy())

	ts, energy := restored.GetPlanEnergy()
	assert.Equal(t, planTime, ts)
	assert.Equal(t, 7.0, energy)
}
