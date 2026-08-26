package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSetPriorityStrategy(t *testing.T) {
	site := NewSite()

	// valid: soc
	require.NoError(t, site.SetPriorityStrategy(api.PrioritySoc))
	assert.Equal(t, api.PrioritySoc, site.GetPriorityStrategy())
	v, err := settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PrioritySoc.String(), v, "soc must be persisted")

	// valid: deficit
	require.NoError(t, site.SetPriorityStrategy(api.PriorityDeficit))
	assert.Equal(t, api.PriorityDeficit, site.GetPriorityStrategy())
	v, err = settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityDeficit.String(), v)

	// valid: none (default)
	require.NoError(t, site.SetPriorityStrategy(api.PriorityNone))
	assert.Equal(t, api.PriorityNone, site.GetPriorityStrategy())
	v, err = settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityNone.String(), v)

	// invalid: rejected, state unchanged
	require.NoError(t, site.SetPriorityStrategy(api.PrioritySoc))
	assert.Error(t, site.SetPriorityStrategy(api.PriorityStrategy(99)))
	assert.Equal(t, api.PrioritySoc, site.GetPriorityStrategy(), "invalid strategy must not change state")
	v, err = settings.String(keys.PriorityStrategy)
	require.NoError(t, err)
	assert.Equal(t, api.PrioritySoc.String(), v, "invalid strategy must not be persisted")
}

func TestSetPriorityBasis(t *testing.T) {
	site := NewSite()

	// valid: energy
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))
	assert.Equal(t, api.PriorityBasisEnergy, site.GetPriorityBasis())
	v, err := settings.String(keys.PriorityBasis)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityBasisEnergy.String(), v, "energy must be persisted")

	// valid: percent (default)
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisPercent))
	assert.Equal(t, api.PriorityBasisPercent, site.GetPriorityBasis())
	v, err = settings.String(keys.PriorityBasis)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityBasisPercent.String(), v)

	// invalid: rejected, state unchanged
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))
	assert.Error(t, site.SetPriorityBasis(api.PriorityBasis(99)))
	assert.Equal(t, api.PriorityBasisEnergy, site.GetPriorityBasis(), "invalid basis must not change state")
	v, err = settings.String(keys.PriorityBasis)
	require.NoError(t, err)
	assert.Equal(t, api.PriorityBasisEnergy.String(), v, "invalid basis must not be persisted")
}

func TestSetPriorityHysteresis(t *testing.T) {
	site := NewSite()

	// valid
	require.NoError(t, site.SetPriorityHysteresis(5))
	assert.Equal(t, 5, site.GetPriorityHysteresis())
	v, err := settings.Int(keys.PriorityHysteresis)
	require.NoError(t, err)
	assert.Equal(t, int64(5), v, "valid hysteresis must be persisted")

	// boundary: 99 ok
	require.NoError(t, site.SetPriorityHysteresis(99))
	assert.Equal(t, 99, site.GetPriorityHysteresis())

	// boundary: 0 ok (off)
	require.NoError(t, site.SetPriorityHysteresis(0))
	assert.Equal(t, 0, site.GetPriorityHysteresis())

	// invalid: > 99 rejected, state unchanged
	require.NoError(t, site.SetPriorityHysteresis(7))
	assert.Error(t, site.SetPriorityHysteresis(100))
	assert.Equal(t, 7, site.GetPriorityHysteresis(), "out-of-range hysteresis must not change state")
	v, err = settings.Int(keys.PriorityHysteresis)
	require.NoError(t, err)
	assert.Equal(t, int64(7), v, "out-of-range hysteresis must not be persisted")

	// invalid: negative rejected
	assert.Error(t, site.SetPriorityHysteresis(-1))
	assert.Equal(t, 7, site.GetPriorityHysteresis(), "negative hysteresis must not change state")
}

// TestEffectivePriorityScoring verifies the site-wide basis/reference resolution: the
// energy basis is only vetoed by a loadpoint that has a comparable soc but no capacity.
func TestEffectivePriorityScoring(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(75.0).AnyTimes()
	vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

	car := NewLoadpoint(util.NewLogger("car"), nil)
	car.vehicleSoc = 20
	car.vehicle = vehicle

	small := api.NewMockVehicle(ctrl)
	small.EXPECT().Capacity().Return(40.0).AnyTimes()
	small.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

	// smaller pack, visited after car: the reference must not follow the last one seen
	second := NewLoadpoint(util.NewLogger("second"), nil)
	second.vehicleSoc = 30
	second.vehicle = small

	// no vehicle and no soc: scores 0 on either basis, hence no veto
	idle := NewLoadpoint(util.NewLogger("idle"), nil)

	// a nil loadpoint (unconfigured slot) must be skipped, not dereferenced

	site := NewSite()
	site.loadpoints = []*Loadpoint{car, second, idle, nil}
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))

	basis, ref := site.EffectivePriorityScoring()
	assert.Equal(t, api.PriorityBasisEnergy, basis)
	assert.Equal(t, 75.0, ref, "reference is the largest capacity in scope")

	// a loadpoint with soc but unknown capacity cannot be ranked in kWh -> percent for all
	unknown := NewLoadpoint(util.NewLogger("unknown"), nil)
	unknown.vehicleSoc = 50
	site.loadpoints = append(site.loadpoints, unknown)

	basis, ref = site.EffectivePriorityScoring()
	assert.Equal(t, api.PriorityBasisPercent, basis)
	assert.Equal(t, 100.0, ref)

	// percent basis always uses the percentage reference
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisPercent))
	basis, ref = site.EffectivePriorityScoring()
	assert.Equal(t, api.PriorityBasisPercent, basis)
	assert.Equal(t, 100.0, ref)
}

// TestPublishedPriorityScoreMatchesRanking verifies that the published score is on the
// same scale the prioritizer ranks with, also when the energy basis is vetoed.
func TestPublishedPriorityScoreMatchesRanking(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(75.0).AnyTimes()
	vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	vehicle.EXPECT().Features().Return(nil).AnyTimes()
	vehicle.EXPECT().Phases().Return(0).AnyTimes()

	car := NewLoadpoint(util.NewLogger("car"), nil)
	car.vehicleSoc = 20
	car.vehicle = vehicle

	// connected vehicle with unknown capacity vetoes the energy basis
	unknown := NewLoadpoint(util.NewLogger("unknown"), nil)
	unknown.vehicleSoc = 50

	site := NewSite()
	site.loadpoints = []*Loadpoint{car, unknown}
	require.NoError(t, site.SetPriorityStrategy(api.PrioritySoc))
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))

	car.site = site
	uiChan := make(chan util.Param, 128)
	car.uiChan = uiChan
	car.PublishEffectiveValues()
	close(uiChan)

	var published any
	for p := range uiChan {
		if p.Key == keys.EffectivePriorityScore {
			published = p.Val
		}
	}

	// the veto ranks by percent: 80% gap -> 0.80, not the raw energy 60kWh -> 0.60
	assert.InDelta(t, 0.80, published, 1e-9)
}

// a heating loadpoint aliases temperature as soc and has no vehicle: it must not veto the
// energy basis, or every site with a heat pump silently falls back to percent
func TestEffectivePriorityScoringHeaterExempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(75.0).AnyTimes()
	vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

	car := NewLoadpoint(util.NewLogger("car"), nil)
	car.vehicleSoc = 20
	car.vehicle = vehicle

	describer := api.NewMockFeatureDescriber(ctrl)
	describer.EXPECT().Features().Return([]api.Feature{api.Heating}).AnyTimes()

	heater := NewLoadpoint(util.NewLogger("heater"), nil)
	heater.vehicleSoc = 55 // temperature, no vehicle
	heater.charger = struct {
		api.Charger
		api.FeatureDescriber
	}{
		Charger:          api.NewMockCharger(ctrl),
		FeatureDescriber: describer,
	}

	site := NewSite()
	site.loadpoints = []*Loadpoint{heater, car}
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))

	basis, ref := site.EffectivePriorityScoring()
	assert.Equal(t, api.PriorityBasisEnergy, basis, "heater must not veto the energy basis")
	assert.InDelta(t, 75.0, ref, 1e-9)
}
