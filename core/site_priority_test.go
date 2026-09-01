package core

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logstash"
	jww "github.com/spf13/jwalterweatherman"
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

// a charger reporting soc itself leaves the loadpoint without a vehicle, which is the
// ordinary way the energy basis gets vetoed. The fallback must be published, or the UI
// keeps labelling the hysteresis in kWh while it is read in percentage points.
func TestEffectivePriorityBasisVetoedByChargerSoc(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(75.0).AnyTimes()
	vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

	car := NewLoadpoint(util.NewLogger("car"), nil)
	car.vehicleSoc = 20
	car.vehicle = vehicle

	battery := api.NewMockBattery(ctrl)
	battery.EXPECT().Soc().Return(50.0, nil).AnyTimes()

	integrated := NewLoadpoint(util.NewLogger("integrated"), nil)
	integrated.title = "wallbox"
	integrated.charger = struct {
		api.Charger
		api.Battery
	}{
		Charger: api.NewMockCharger(ctrl),
		Battery: battery,
	}

	integrated.publishSocAndRange()
	require.Positive(t, integrated.GetSoc(), "charger soc must reach the loadpoint")
	require.Nil(t, integrated.GetVehicle(), "a charger soc assigns no vehicle")

	site := NewSite()
	site.loadpoints = []*Loadpoint{car, integrated}
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))
	require.NoError(t, site.SetPriorityHysteresis(10)) // a deadband that changes unit with the basis

	basis, ref, conflict := site.effectivePriorityScoring()
	assert.Equal(t, api.PriorityBasisPercent, basis, "charger soc without capacity vetoes the energy basis")
	assert.Equal(t, 100.0, ref)
	require.NotNil(t, conflict, "the veto must identify the loadpoint it came from")
	assert.Equal(t, "wallbox", conflict.GetTitle(), "the warning names the offending loadpoint")

	assert.Equal(t, api.PriorityBasisPercent, publishedPriorityBasis(site), "the fallback must be published")
	assert.Equal(t, api.PriorityBasisPercent, publishedPriorityBasis(site), "the fallback holds while it applies")
	assert.Equal(t, api.PriorityBasisEnergy, site.GetPriorityBasis(), "the configured basis stays unchanged")
	assert.Equal(t, conflict, site.priorityBasisConflict, "the offender is remembered to gate the warning")

	// not sticky: the configured basis returns once the charger stops reporting a soc
	integrated.vehicleSoc = 0
	assert.Equal(t, api.PriorityBasisEnergy, publishedPriorityBasis(site))
	assert.Nil(t, site.priorityBasisConflict)
}

// no loadpoint carrying a comparable soc is not a veto: nothing is being ranked, so the
// configured basis must be reported and left alone across unplug/replug
func TestEffectivePriorityBasisIdleKeepsConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(75.0).AnyTimes()
	vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

	// the only car, disconnected: reports no soc
	car := NewLoadpoint(util.NewLogger("car"), nil)
	car.vehicle = vehicle

	site := NewSite()
	site.loadpoints = []*Loadpoint{car}
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))

	_, _, conflict := site.effectivePriorityScoring()
	assert.Nil(t, conflict, "an idle site conflicts with nothing")

	assert.Equal(t, api.PriorityBasisEnergy, publishedPriorityBasis(site), "idle must not report the percent fallback")
	assert.Nil(t, site.priorityBasisConflict, "an idle site must not log a veto")

	// plug in: the basis is honoured, so the published value must not move
	car.vehicleSoc = 20
	assert.Equal(t, api.PriorityBasisEnergy, publishedPriorityBasis(site))
	assert.Nil(t, site.priorityBasisConflict)

	// unplug again: still no flip, or the modal label oscillates kWh -> % -> kWh
	car.vehicleSoc = 0
	assert.Equal(t, api.PriorityBasisEnergy, publishedPriorityBasis(site))
	assert.Nil(t, site.priorityBasisConflict)
}

// publishedPriorityBasis runs one publish cycle and returns the effective basis it published
func publishedPriorityBasis(site *Site) any {
	ch := make(chan util.Param, 8)
	site.valueChan = ch
	site.publishPriorityBasis()
	close(ch)

	var val any
	for p := range ch {
		if p.Key == keys.EffectivePriorityBasis {
			val = p.Val
		}
	}
	return val
}

// the warning is gated on the offending loadpoint, not on a flag: a persistent conflict
// must stay quiet, but a new offender must re-warn instead of leaving the log blaming a
// loadpoint that has since disconnected
func TestEffectivePriorityBasisRewarnsOnNewConflict(t *testing.T) {
	ctrl := gomock.NewController(t)

	socReportingCharger := func(title string) *Loadpoint {
		battery := api.NewMockBattery(ctrl)
		battery.EXPECT().Soc().Return(50.0, nil).AnyTimes()

		lp := NewLoadpoint(util.NewLogger(title), nil)
		lp.title = title
		lp.charger = struct {
			api.Charger
			api.Battery
		}{
			Charger: api.NewMockCharger(ctrl),
			Battery: battery,
		}
		lp.publishSocAndRange()
		return lp
	}

	first := socReportingCharger("wallboxA")
	second := socReportingCharger("wallboxB")

	site := NewSite()
	site.loadpoints = []*Loadpoint{first, second}
	require.NoError(t, site.SetPriorityBasis(api.PriorityBasisEnergy))

	warnings := func(title string) int {
		var n int
		for _, e := range logstash.All(nil, jww.LevelWarn, 0) {
			if strings.Contains(e, "priority basis") && strings.Contains(e, title) {
				n++
			}
		}
		return n
	}

	publishedPriorityBasis(site)
	assert.Equal(t, 1, warnings("wallboxA"), "the first conflict must warn")
	assert.Equal(t, 0, warnings("wallboxB"))

	// same offender across further cycles: no repeat
	publishedPriorityBasis(site)
	publishedPriorityBasis(site)
	assert.Equal(t, 1, warnings("wallboxA"), "a persistent conflict must not re-log every cycle")

	// the first loadpoint disconnects, the second is now the offender
	first.vehicleSoc = 0
	publishedPriorityBasis(site)
	assert.Equal(t, 1, warnings("wallboxB"), "a new offender must re-warn")
	assert.Equal(t, 1, warnings("wallboxA"), "the retired offender must not be blamed again")
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
