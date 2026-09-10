package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type featureCharger struct {
	api.Charger
	features []api.Feature
}

func (c *featureCharger) Features() []api.Feature { return c.features }

func TestSetModeLegacyAliases(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))

	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	// minpv maps to smart and enables always charge
	lp.SetMode(api.ModeMinPV)
	assert.Equal(t, api.ModeSmart, lp.GetMode())
	assert.Equal(t, api.AlwaysChargeOn, lp.GetAlwaysCharge())

	// smart leaves always charge untouched
	lp.SetMode(api.ModeOff)
	lp.SetMode(api.ModeSmart)
	assert.Equal(t, api.AlwaysChargeOn, lp.GetAlwaysCharge())

	// pv maps to smart and disables always charge, even when mode is already smart
	lp.SetMode(api.ModePV)
	assert.Equal(t, api.ModeSmart, lp.GetMode())
	assert.Equal(t, api.AlwaysChargeOff, lp.GetAlwaysCharge())

	// mode change does not clear always charge
	require.NoError(t, lp.SetAlwaysCharge(api.AlwaysChargeOnce))
	lp.SetMode(api.ModeNow)
	assert.Equal(t, api.AlwaysChargeOnce, lp.GetAlwaysCharge())
}

func TestSetModeLegacyAliasesSwitchDevice(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))
	lp.charger = &featureCharger{features: []api.Feature{api.SwitchDevice}}

	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	// no current control: minpv maps to smart without always charge
	lp.SetMode(api.ModeMinPV)
	assert.Equal(t, api.ModeSmart, lp.GetMode())
	assert.Equal(t, api.AlwaysChargeOff, lp.GetAlwaysCharge())

	assert.Error(t, lp.SetAlwaysCharge(api.AlwaysChargeOn))
}

func TestAlwaysChargeOnceResetsOnDisconnect(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))

	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	require.NoError(t, lp.SetAlwaysCharge(api.AlwaysChargeOnce))
	lp.evVehicleDisconnectHandler()
	assert.Equal(t, api.AlwaysChargeOff, lp.GetAlwaysCharge())

	require.NoError(t, lp.SetAlwaysCharge(api.AlwaysChargeOn))
	lp.evVehicleDisconnectHandler()
	assert.Equal(t, api.AlwaysChargeOn, lp.GetAlwaysCharge())
}

func TestNormalizeMode(t *testing.T) {
	switchDevice := &featureCharger{features: []api.Feature{api.SwitchDevice}}
	continuous := &featureCharger{features: []api.Feature{api.Continuous}}

	for _, tc := range []struct {
		name    string
		charger api.Charger
		in      api.ChargeMode
		mode    api.ChargeMode
		ac      api.AlwaysCharge
	}{
		{"minpv", nil, api.ModeMinPV, api.ModeSmart, api.AlwaysChargeOn},
		{"pv", nil, api.ModePV, api.ModeSmart, api.AlwaysChargeOff},
		{"smart untouched", nil, api.ModeSmart, api.ModeSmart, ""},
		{"now untouched", nil, api.ModeNow, api.ModeNow, ""},
		{"minpv switch device", switchDevice, api.ModeMinPV, api.ModeSmart, ""},
		{"pv switch device", switchDevice, api.ModePV, api.ModeSmart, ""},
		{"minpv continuous", continuous, api.ModeMinPV, api.ModeSmart, api.AlwaysChargeOn},
	} {
		lp := &Loadpoint{charger: tc.charger}
		mode, ac := lp.normalizeMode(tc.in)
		assert.Equal(t, tc.mode, mode, tc.name)
		assert.Equal(t, tc.ac, ac, tc.name)
	}
}

func TestSetDefaultModeLegacyAliases(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))

	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	// deprecated defaults map to smart without touching always charge
	for _, mode := range []api.ChargeMode{api.ModeMinPV, api.ModePV} {
		lp.SetDefaultMode(mode)
		assert.Equal(t, api.ModeSmart, lp.GetDefaultMode())
		assert.Equal(t, api.AlwaysChargeOff, lp.GetAlwaysCharge())
	}
}

func TestLegacyDefaultModeSeedsAlwaysCharge(t *testing.T) {
	ctrl := gomock.NewController(t)
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{Name: "seed-charger"}, api.Charger(api.NewMockCharger(ctrl)))))

	dbSettings := settings.NewDatabaseSettingsAdapter("seed.")
	create := func() *Loadpoint {
		lp, err := NewLoadpointFromConfig(util.NewLogger("foo"), dbSettings, nil, map[string]any{
			"charger": "seed-charger",
			"mode":    "minpv",
		})
		require.NoError(t, err)
		return lp
	}

	// first boot: legacy default seeds and persists always charge, default becomes smart
	lp := create()
	assert.Equal(t, api.ModeSmart, lp.GetMode())
	assert.Equal(t, api.ModeSmart, lp.GetDefaultMode())
	assert.Equal(t, api.AlwaysChargeOn, lp.GetAlwaysCharge())
	v, err := dbSettings.String(keys.AlwaysCharge)
	require.NoError(t, err)
	assert.Equal(t, "on", v)

	// later boots: the persisted user choice wins over the seed
	dbSettings.SetString(keys.AlwaysCharge, "off")
	lp = create()
	assert.Equal(t, api.AlwaysChargeOff, lp.GetAlwaysCharge())
	v, _ = dbSettings.String(keys.AlwaysCharge)
	assert.Equal(t, "off", v)
}
