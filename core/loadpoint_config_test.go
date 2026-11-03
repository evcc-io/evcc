package core

import (
	"maps"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// blobSettings mirrors settings.ConfigSettings: writes additionally land in the
// loadpoint's own config blob instead of a separate settings table.
type blobSettings struct {
	settings.Settings
	blob map[string]any
}

func newBlobSettings() *blobSettings {
	return &blobSettings{Settings: settings.NewMemorySettings(), blob: make(map[string]any)}
}

func (s *blobSettings) SetString(key string, val string) {
	s.blob[key] = val
	s.Settings.SetString(key, val)
}

func (s *blobSettings) SetInt(key string, val int64) {
	s.blob[key] = val
	s.Settings.SetInt(key, val)
}

func (s *blobSettings) SetFloat(key string, val float64) {
	s.blob[key] = val
	s.Settings.SetFloat(key, val)
}

func (s *blobSettings) SetFloatPtr(key string, val *float64) {
	s.blob[key] = val
	s.Settings.SetFloatPtr(key, val)
}

func (s *blobSettings) SetTime(key string, val time.Time) {
	s.blob[key] = val
	s.Settings.SetTime(key, val)
}

func (s *blobSettings) SetBool(key string, val bool) {
	s.blob[key] = val
	s.Settings.SetBool(key, val)
}

func (s *blobSettings) SetJson(key string, val any) error {
	s.blob[key] = val
	return s.Settings.SetJson(key, val)
}

// TestPersistedSettingsAreValidConfig asserts that settings of a database-configured loadpoint,
// persisted into its own config blob, remain decodable as loadpoint config on the next boot.
func TestPersistedSettingsAreValidConfig(t *testing.T) {
	store := newBlobSettings()
	uiChan, pushChan, lpChan := createChannels(t)

	planTime := time.Now().Add(4 * time.Hour).Round(time.Second)
	limit := 0.25

	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	charger.EXPECT().Enabled().Return(false, nil).AnyTimes()

	lp := NewLoadpoint(util.NewLogger("foo"), store)
	lp.charger = charger
	lp.Prepare(new(Site), uiChan, pushChan, lpChan)

	// exercise the persisting setters
	lp.SetTitle("title")
	lp.SetMode(api.ModeSmart)
	lp.SetDefaultMode(api.ModeSmart)
	require.NoError(t, lp.SetAlwaysCharge(api.AlwaysChargeOn))
	lp.SetPriority(3)
	require.NoError(t, lp.SetMinCurrent(8))
	require.NoError(t, lp.SetMaxCurrent(12))
	require.NoError(t, lp.SetPhasesConfigured(1))
	lp.SetLimitSoc(75)
	lp.SetLimitEnergy(11)
	lp.SetMinSoc(20)
	lp.SetSmartCostLimit(&limit)
	lp.SetSmartFeedInPriorityLimit(&limit)
	lp.SetBatteryBoostLimit(90)
	lp.SetSolarShare(0.6)
	lp.SetThresholds(loadpoint.ThresholdsConfig{
		Enable:  loadpoint.ThresholdConfig{Delay: 2 * time.Minute, Threshold: -100},
		Disable: loadpoint.ThresholdConfig{Delay: 4 * time.Minute, Threshold: 200},
	})
	lp.SetSocConfig(loadpoint.SocConfig{
		Poll: loadpoint.PollConfig{Mode: loadpoint.PollAlways, Interval: 42 * time.Minute},
	})
	lp.SetUI(loadpoint.UIConfig{MinTemp: 20, MaxTemp: 45})
	require.NoError(t, lp.SetPlanEnergy(planTime, 7))
	require.NoError(t, lp.SetPlanStrategy(api.PlanStrategy{Continuous: true}))

	require.NotEmpty(t, store.blob)

	// boot path: split the config blob and decode the static remainder strictly
	_, static, err := loadpoint.SplitConfig(maps.Clone(store.blob))
	require.NoError(t, err)
	require.NoError(t, util.DecodeOther(static, new(Loadpoint)), "persisted setting is not valid loadpoint config")
}
