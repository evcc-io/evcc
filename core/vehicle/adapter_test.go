package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestAdapterSetModeLegacyAliases(t *testing.T) {
	v := &adapter{log: util.NewLogger("foo"), name: "aliases"}

	// deprecated modes map to smart without touching always charge
	for _, mode := range []api.ChargeMode{api.ModeMinPV, api.ModePV} {
		v.SetMode(mode)
		assert.Equal(t, api.ModeSmart, v.GetMode())
		assert.Equal(t, api.AlwaysCharge(""), v.GetAlwaysCharge())
	}

	v.SetAlwaysCharge(api.AlwaysChargeOn)
	v.SetMode(api.ModeNow)
	assert.Equal(t, api.ModeNow, v.GetMode())
	assert.Equal(t, api.AlwaysChargeOn, v.GetAlwaysCharge(), "mode change keeps always charge")

	v.SetAlwaysCharge("")
	assert.Equal(t, api.AlwaysCharge(""), v.GetAlwaysCharge())
}

func TestAdapterGetModeMigratesLegacy(t *testing.T) {
	for _, tc := range []struct {
		stored string
		ac     api.AlwaysCharge
	}{
		{"minpv", api.AlwaysChargeOn},
		{"pv", api.AlwaysChargeOff},
	} {
		v := &adapter{log: util.NewLogger("foo"), name: "legacy-" + tc.stored}
		settings.SetString(v.key()+keys.Mode, tc.stored)

		assert.Equal(t, api.ModeSmart, v.GetMode(), tc.stored)
		assert.Equal(t, tc.ac, v.GetAlwaysCharge(), tc.stored)

		s, err := settings.String(v.key() + keys.Mode)
		assert.NoError(t, err)
		assert.Equal(t, "smart", s, "migration persisted")
	}
}
