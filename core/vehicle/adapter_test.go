package vehicle

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapterSetModeLegacyAliases(t *testing.T) {
	v := &adapter{log: util.NewLogger("foo"), name: "aliases"}

	// deprecated modes map to smart and carry the equivalent always charge
	for _, tc := range []struct {
		mode api.ChargeMode
		ac   api.AlwaysCharge
	}{
		{api.ModeMinPV, api.AlwaysChargeOn},
		{api.ModePV, api.AlwaysChargeOff},
	} {
		v.SetMode(tc.mode)
		assert.Equal(t, api.ModeSmart, v.GetMode(), tc.mode)
		assert.Equal(t, tc.ac, v.GetAlwaysCharge(), tc.mode)
	}

	v.SetAlwaysCharge(api.AlwaysChargeOn)
	v.SetMode(api.ModeNow)
	assert.Equal(t, api.ModeNow, v.GetMode())
	assert.Equal(t, api.AlwaysChargeOn, v.GetAlwaysCharge(), "mode change keeps always charge")

	v.SetAlwaysCharge("")
	assert.Equal(t, api.AlwaysCharge(""), v.GetAlwaysCharge())
}

func TestAdapterGetModeMigratesLegacy(t *testing.T) {
	// site publish re-reads every vehicle mode; migration must not recurse
	var v *adapter
	Publish = func() { v.GetMode() }
	t.Cleanup(func() { Publish = nil })

	for _, tc := range []struct {
		stored string
		ac     api.AlwaysCharge
	}{
		{"minpv", api.AlwaysChargeOn},
		{"pv", api.AlwaysChargeOff},
	} {
		v = &adapter{log: util.NewLogger("foo"), name: "legacy-" + tc.stored}
		settings.SetString(v.key()+keys.Mode, tc.stored)

		assert.Equal(t, api.ModeSmart, v.GetMode(), tc.stored)
		assert.Equal(t, tc.ac, v.GetAlwaysCharge(), tc.stored)

		s, err := settings.String(v.key() + keys.Mode)
		assert.NoError(t, err)
		assert.Equal(t, "smart", s, "migration persisted")
	}
}

func TestAdapterSetOneTimePlans(t *testing.T) {
	v := &adapter{log: util.NewLogger("foo"), name: "onetime"}
	t.Cleanup(func() { _ = settings.Delete(v.key() + keys.OneTimePlans) })

	later, sooner := time.Now().Add(48*time.Hour).Round(time.Second), time.Now().Add(24*time.Hour).Round(time.Second)

	// stored ordered by time
	require.NoError(t, v.SetOneTimePlans([]api.OneTimePlan{{Time: later, Soc: 65}, {Time: sooner, Soc: 100}}))
	plans := v.GetOneTimePlans()
	require.Len(t, plans, 2)
	assert.Equal(t, 100, plans[0].Soc)
	assert.Equal(t, 65, plans[1].Soc)

	// invalid input is rejected and leaves stored plans untouched
	assert.Error(t, v.SetOneTimePlans([]api.OneTimePlan{{Time: later, Soc: 0}}), "soc zero")
	assert.Error(t, v.SetOneTimePlans([]api.OneTimePlan{{Time: later, Soc: 101}}), "soc above 100")
	assert.Len(t, v.GetOneTimePlans(), 2)

	// expired plans are dropped
	require.NoError(t, v.SetOneTimePlans([]api.OneTimePlan{{Time: later, Soc: 65}, {Time: time.Now().Add(-time.Hour), Soc: 80}}))
	assert.Len(t, v.GetOneTimePlans(), 1)

	// empty list clears
	require.NoError(t, v.SetOneTimePlans(nil))
	assert.Empty(t, v.GetOneTimePlans())
}
