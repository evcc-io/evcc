package core

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	optimizer "github.com/evcc-io/optimizer/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// curtailableMeter counts device interactions to verify caching.
type curtailableMeter struct {
	api.Meter
	percent  int
	getErr   error
	setErr   error
	gets     int
	setCalls []int
}

func (m *curtailableMeter) CurtailedPercent() (int, error) {
	m.gets++
	return m.percent, m.getErr
}

func (m *curtailableMeter) SetCurtailPercent(percent int) error {
	m.setCalls = append(m.setCalls, percent)
	if m.setErr != nil {
		return m.setErr
	}
	m.percent = percent
	return nil
}

func curtailSite(m api.Meter) *Site {
	return &Site{
		log:      util.NewLogger("foo"),
		pvMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, m)},
	}
}

// The HEMS percent is applied once and not re-evaluated while it stays unchanged.
func TestCurtailPVCache(t *testing.T) {
	m := &curtailableMeter{percent: 100}
	site := curtailSite(m)

	require.NoError(t, site.curtailPV(new(60)))
	assert.Equal(t, []int{60}, m.setCalls)
	assert.Equal(t, 1, m.gets)

	// unchanged: device is not queried again
	for range 3 {
		require.NoError(t, site.curtailPV(new(60)))
	}
	assert.Equal(t, []int{60}, m.setCalls)
	assert.Equal(t, 1, m.gets)

	// changed: applied again. A bool device state could not distinguish 60 from 30
	require.NoError(t, site.curtailPV(new(30)))
	assert.Equal(t, []int{60, 30}, m.setCalls)

	require.NoError(t, site.curtailPV(new(100)))
	assert.Equal(t, []int{60, 30, 100}, m.setCalls)
}

// A failed write must be retried on the next cycle instead of being cached.
func TestCurtailPVCacheRetriesAfterError(t *testing.T) {
	m := &curtailableMeter{percent: 100, setErr: errors.New("nope")}
	site := curtailSite(m)

	require.Error(t, site.curtailPV(new(60)))
	require.Error(t, site.curtailPV(new(60)))
	assert.Equal(t, []int{60, 60}, m.setCalls)

	m.setErr = nil
	require.NoError(t, site.curtailPV(new(60)))
	require.NoError(t, site.curtailPV(new(60)))
	assert.Equal(t, []int{60, 60, 60}, m.setCalls)
}

// nil percent means the HEMS makes no statement and must not touch the cache.
func TestCurtailPVNoStatement(t *testing.T) {
	m := &curtailableMeter{percent: 100}
	site := curtailSite(m)

	require.NoError(t, site.curtailPV(new(60)))
	require.NoError(t, site.curtailPV(nil))
	require.NoError(t, site.curtailPV(new(60)))
	assert.Equal(t, []int{60}, m.setCalls)
}

// A device that cannot report its state is written once, not on every cycle.
func TestCurtailPVNotAvailable(t *testing.T) {
	m := &curtailableMeter{getErr: api.ErrNotAvailable}
	site := curtailSite(m)

	for range 3 {
		require.NoError(t, site.curtailPV(new(60)))
	}
	assert.Equal(t, []int{60}, m.setCalls)

	// changed: applied again
	require.NoError(t, site.curtailPV(new(30)))
	assert.Equal(t, []int{60, 30}, m.setCalls)
}

// A curtailment device is applied like a curtailable pv meter.
func TestCurtailDevice(t *testing.T) {
	m := &curtailableMeter{percent: 100}
	site := &Site{
		log:        util.NewLogger("foo"),
		curtailers: []config.Device[api.Curtailer]{config.NewStaticDevice[api.Curtailer](config.Named{}, m)},
	}

	require.NoError(t, site.curtailPV(new(60)))
	assert.Equal(t, []int{60}, m.setCalls)
	assert.Equal(t, 1, m.gets)

	// unchanged: device is not queried again
	require.NoError(t, site.curtailPV(new(60)))
	assert.Equal(t, []int{60}, m.setCalls)
	assert.Equal(t, 1, m.gets)
}

// dimmableMeter counts device interactions to verify caching.
type dimmableMeter struct {
	api.Meter
	dimmed   bool
	getErr   error
	dimErr   error
	gets     int
	dimCalls []bool
}

func (m *dimmableMeter) Dimmed() (bool, error) {
	m.gets++
	return m.dimmed, m.getErr
}

func (m *dimmableMeter) Dim(dim bool) error {
	m.dimCalls = append(m.dimCalls, dim)
	if m.dimErr != nil {
		return m.dimErr
	}
	m.dimmed = dim
	return nil
}

func dimSite(m api.Meter) *Site {
	return &Site{
		log:       util.NewLogger("foo"),
		auxMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, m)},
	}
}

func TestDimMetersCache(t *testing.T) {
	m := &dimmableMeter{}
	site := dimSite(m)

	require.NoError(t, site.dimMeters(true))
	assert.Equal(t, []bool{true}, m.dimCalls)
	assert.Equal(t, 1, m.gets)

	for range 3 {
		require.NoError(t, site.dimMeters(true))
	}
	assert.Equal(t, []bool{true}, m.dimCalls)
	assert.Equal(t, 1, m.gets)

	require.NoError(t, site.dimMeters(false))
	assert.Equal(t, []bool{true, false}, m.dimCalls)

	// failed write is retried
	m.dimErr = errors.New("nope")
	require.Error(t, site.dimMeters(true))
	require.Error(t, site.dimMeters(true))
	assert.Equal(t, []bool{true, false, true, true}, m.dimCalls)
}

// A device that cannot report its state is written once, not on every cycle.
func TestDimMetersNotAvailable(t *testing.T) {
	m := &dimmableMeter{getErr: api.ErrNotAvailable}
	site := dimSite(m)

	for range 3 {
		require.NoError(t, site.dimMeters(true))
	}
	assert.Equal(t, []bool{true}, m.dimCalls)

	require.NoError(t, site.dimMeters(false))
	assert.Equal(t, []bool{true, false}, m.dimCalls)
}

// The curtail percent scales with the export beyond the limit and converges at the cap.
func TestOptimizerCurtailPercent(t *testing.T) {
	for _, tc := range []struct {
		name            string
		percent         int
		grid, pv, limit float64
		expect          int
	}{
		{"export above limit", 100, -7000, 10000, 5000, 80},
		{"at the cap, still above limit", 80, -6000, 8000, 5000, 70},
		{"below limit raises", 70, -4000, 7000, 5000, 80},
		{"within threshold", 70, -5030, 7000, 5000, 70},
		{"no pv", 70, -7000, 0, 5000, 70},
		{"clamped to 0", 100, -20000, 10000, 5000, 0},
		{"clamped to 100", 90, 1000, 3000, 5000, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, optimizerCurtailPercent(tc.percent, tc.grid, tc.pv, tc.limit))
		})
	}
}

// The optimizer curtails while the active slot plans overshoot and releases its curtailment afterwards.
func optimizerCurtailSite(t *testing.T, m *curtailableMeter, now time.Time) *Site {
	t.Helper()

	site := curtailSite(m)
	site.gridMeter = config.NewStaticDevice(config.Named{}, api.Meter(m))
	site.gridExportLimit = 5000
	site.lastOptimizerSolve = &optimizerSolve{
		schedule: optimizerSchedule{timestamps: []time.Time{now}, dt: []int{900}},
		res:      optimizer.OptimizationResult{GridExportOvershoot: []float32{250}},
	}

	return site
}

func TestApplyOptimizerCurtailment(t *testing.T) {
	enableAutomatic(t)

	now := time.Now()
	m := &curtailableMeter{percent: 100}
	site := optimizerCurtailSite(t, m, now)

	site.applyOptimizerCurtailment(-7000, 10000, now)
	assert.Equal(t, []int{80}, m.setCalls)
	assert.True(t, site.optimizerCurtailed)

	// no planned overshoot: release
	site.lastOptimizerSolve = nil
	site.applyOptimizerCurtailment(-4000, 8000, now)
	assert.Equal(t, []int{80, 100}, m.setCalls)
	assert.False(t, site.optimizerCurtailed)

	// not curtailed by the optimizer: left alone
	site.applyOptimizerCurtailment(-4000, 8000, now)
	assert.Equal(t, []int{80, 100}, m.setCalls)
}

// An uncurtailed HEMS statement leaves the pv to the optimizer, an active one takes precedence.
func TestApplyOptimizerCurtailmentHems(t *testing.T) {
	enableAutomatic(t)

	now := time.Now()
	m := &curtailableMeter{percent: 100}
	site := optimizerCurtailSite(t, m, now)

	h := api.NewMockHEMS(gomock.NewController(t))
	site.hems = h

	h.EXPECT().CurtailedPercent().Return(new(100)).AnyTimes()
	h.EXPECT().MaxConsumptionPower().Return(nil).AnyTimes()
	for range 2 {
		site.applyHemsLimits()
		site.applyOptimizerCurtailment(-7000, 10000, now)
	}
	assert.Equal(t, []int{80}, m.setCalls[:1])
	assert.NotContains(t, m.setCalls[1:], 100, "hems must not reset optimizer curtailment")

	h = api.NewMockHEMS(gomock.NewController(t))
	site.hems = h
	h.EXPECT().CurtailedPercent().Return(new(60)).AnyTimes()
	h.EXPECT().MaxConsumptionPower().Return(nil).AnyTimes()
	site.applyHemsLimits()
	site.applyOptimizerCurtailment(-7000, 10000, now)
	assert.Equal(t, 60, m.percent)
	assert.False(t, site.optimizerCurtailed)
}

// A partially failed curtailment is released when the plan ends.
func TestApplyOptimizerCurtailmentPartial(t *testing.T) {
	enableAutomatic(t)

	now := time.Now()
	ok := &curtailableMeter{percent: 100}
	failing := &curtailableMeter{percent: 100, setErr: errors.New("nope")}
	site := optimizerCurtailSite(t, ok, now)
	site.pvMeters = append(site.pvMeters, config.NewStaticDevice(config.Named{}, api.Meter(failing)))

	site.applyOptimizerCurtailment(-7000, 10000, now)
	assert.Equal(t, 80, ok.percent)

	failing.setErr = nil
	site.lastOptimizerSolve = nil
	site.applyOptimizerCurtailment(-4000, 8000, now)
	assert.Equal(t, 100, ok.percent)
	assert.False(t, site.optimizerCurtailed)
}
