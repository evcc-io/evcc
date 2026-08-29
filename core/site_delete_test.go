package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// deleteTestSite mirrors what Prepare wires up: cached devices plus one energy
// collector per meter, keyed by the meter's config name.
type deleteTestSite struct {
	*Site
	params chan util.Param
}

func newDeleteTestSite(t *testing.T, ctrl *gomock.Controller) *deleteTestSite {
	t.Helper()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, metrics.SetupSchema())

	meter := func(power float64) api.Meter {
		m := api.NewMockMeter(ctrl)
		m.EXPECT().CurrentPower().Return(power, nil).AnyTimes()
		return m
	}

	battery := func(power, soc float64) api.Meter {
		m := api.NewMockMeter(ctrl)
		m.EXPECT().CurrentPower().Return(power, nil).AnyTimes()
		b := api.NewMockBattery(ctrl)
		b.EXPECT().Soc().Return(soc, nil).AnyTimes()
		return &struct {
			api.Meter
			api.Battery
		}{Meter: m, Battery: b}
	}

	dev := func(name string, m api.Meter) config.Device[api.Meter] {
		return config.NewStaticDevice(config.Named{Name: name}, m)
	}

	params := make(chan util.Param, 1024)

	site := NewSite()
	site.log = util.NewLogger("foo")
	site.valueChan = params

	site.gridMeter = dev("grid", meter(1e3))
	site.pvMeters = []config.Device[api.Meter]{dev("pv", meter(2e3))}
	site.batteryMeters = []config.Device[api.Meter]{dev("battery", battery(-500, 42))}

	// collectors as Prepare registers them: keyed by the meter's config name
	for _, c := range []struct{ group, name string }{
		{metrics.Grid, "grid"},
		{metrics.PV, "pv"},
		{metrics.Battery, "battery"},
		{metrics.Forecast, metrics.Forecast},
		{metrics.Temperature, metrics.Temperature},
	} {
		col, err := metrics.NewCollector(c.group, c.name, c.name)
		require.NoError(t, err)
		site.collectors[c.name] = col
	}

	// site settings pointing at those meters
	site.Meters.GridMeterRef = "grid"
	site.Meters.PVMetersRef = []string{"pv"}
	site.Meters.BatteryMetersRef = []string{"battery"}

	return &deleteTestSite{Site: site, params: params}
}

// published drains the value channel and returns the last value per key
func (s *deleteTestSite) published() map[string]any {
	res := make(map[string]any)
	for {
		select {
		case p := <-s.params:
			res[p.Key] = p.Val
		default:
			return res
		}
	}
}

// TestSiteUpdateAfterMeterDeletion deletes the active grid, pv and battery meters while
// running. Only the refs are cleared, the cached devices keep being polled (#32605).
func TestSiteUpdateAfterMeterDeletion(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newDeleteTestSite(t, ctrl)

	// baseline: everything still referenced
	require.NoError(t, s.updateGridMeter())
	s.updatePvMeters()
	s.updateBatteryMeters()

	grid, ok := s.published()[keys.Grid].(types.Measurement)
	require.True(t, ok, "grid measurement published")
	assert.Equal(t, "grid", grid.Name, "grid measurement carries the meter name")

	// delete all three meters- exactly what deleteDeviceHandler does to the site
	s.SetGridMeterRef("")
	s.SetPVMeterRefs(nil)
	s.SetBatteryMeterRefs(nil)

	require.NotPanics(t, func() {
		require.NoError(t, s.updateGridMeter())
		s.updatePvMeters()
		s.updateBatteryMeters()
	}, "update loop must survive the refs being cleared")

	// the devices are still cached, so the loop keeps producing measurements
	pub := s.published()
	assert.Equal(t, 2e3, s.pvPower, "pv still polled")
	assert.Equal(t, -500.0, s.battery.Power, "battery still polled")

	grid, ok = pub[keys.Grid].(types.Measurement)
	require.True(t, ok, "grid measurement still published")
	assert.Equal(t, "grid", grid.Name, "grid measurement must keep the meter name, not the cleared ref")
}

// TestSiteUpdateAfterTariffDeletion deletes a tariff device. cleanupTariffRef only
// rewrites the persisted refs, so the publish path must cope either way.
func TestSiteUpdateAfterTariffDeletion(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newDeleteTestSite(t, ctrl)

	solar := api.NewMockTariff(ctrl)
	solar.EXPECT().Type().Return(api.TariffTypeSolar).AnyTimes()
	solar.EXPECT().Rates().Return(api.Rates{
		{Start: time.Now().Add(-time.Hour), End: time.Now(), Value: 1e3},
		{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 2e3},
	}, nil).AnyTimes()

	s.tariffs = &tariff.Tariffs{Solar: solar}

	require.NotPanics(t, func() {
		s.publishTariffs(0, 0)
	}, "tariff publish with a solar tariff")

	require.NotNil(t, s.published()[keys.Forecast], "forecast published while the tariff exists")

	// delete the tariff
	s.tariffs = &tariff.Tariffs{}

	require.NotPanics(t, func() {
		s.publishTariffs(0, 0)
	}, "tariff publish must survive the tariff being deleted")
}
