package core

import (
	"errors"
	"math"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (m *dimmableMeter) DimPower() float64 {
	return 0
}

func (m *dimmableMeter) Dim(dim bool, _ float64) error {
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

	require.NoError(t, site.dimMeters(dimState{active: true}))
	assert.Equal(t, []bool{true}, m.dimCalls)
	assert.Equal(t, 1, m.gets)

	for range 3 {
		require.NoError(t, site.dimMeters(dimState{active: true}))
	}
	assert.Equal(t, []bool{true}, m.dimCalls)
	assert.Equal(t, 1, m.gets)

	require.NoError(t, site.dimMeters(dimState{}))
	assert.Equal(t, []bool{true, false}, m.dimCalls)

	// failed write is retried
	m.dimErr = errors.New("nope")
	require.Error(t, site.dimMeters(dimState{active: true}))
	require.Error(t, site.dimMeters(dimState{active: true}))
	assert.Equal(t, []bool{true, false, true, true}, m.dimCalls)
}

// A device that cannot report its state is written once, not on every cycle.
func TestDimMetersNotAvailable(t *testing.T) {
	m := &dimmableMeter{getErr: api.ErrNotAvailable}
	site := dimSite(m)

	for range 3 {
		require.NoError(t, site.dimMeters(dimState{active: true}))
	}
	assert.Equal(t, []bool{true}, m.dimCalls)

	require.NoError(t, site.dimMeters(dimState{}))
	assert.Equal(t, []bool{true, false}, m.dimCalls)
}

// A dimmer appearing later receives the current state although the state is unchanged.
func TestDimMetersNewDimmer(t *testing.T) {
	m1 := &dimmableMeter{}
	site := dimSite(m1)

	require.NoError(t, site.dimMeters(dimState{active: true}))
	assert.Equal(t, []bool{true}, m1.dimCalls)

	m2 := &dimmableMeter{}
	site.auxMeters = append(site.auxMeters, config.NewStaticDevice[api.Meter](config.Named{}, m2))

	require.NoError(t, site.dimMeters(dimState{active: true}))
	assert.Equal(t, []bool{true}, m2.dimCalls)
}

type dimPowerMeter struct {
	dimmableMeter
	power float64
}

func (m *dimPowerMeter) DimPower() float64 {
	return m.power
}

func TestDimPowerInvalid(t *testing.T) {
	for _, p := range []float64{-1, math.NaN(), math.Inf(1)} {
		assert.Zero(t, dimPower(&dimPowerMeter{power: p}), p)
		assert.Zero(t, dimState{active: true}.limit(&dimPowerMeter{power: p}), p)
	}
	assert.Equal(t, 4200.0, dimPower(&dimPowerMeter{power: 4200}))
}

// A negative dim power exempts the device: never dimmed, released if dimmed, not counted.
func TestDimMetersExempt(t *testing.T) {
	m := &dimPowerMeter{power: -1}
	site := dimSite(m)

	require.NoError(t, site.dimMeters(dimState{active: true}))
	assert.Empty(t, m.dimCalls)

	m.dimmed = true
	site.dimmed = nil
	require.NoError(t, site.dimMeters(dimState{active: true}))
	assert.Equal(t, []bool{false}, m.dimCalls)

	assert.False(t, dimState{active: true}.dimActive(m))
	assert.True(t, dimState{active: true}.dimActive(&dimPowerMeter{power: 0}))
}

// Disabled loadpoints are kept as nil entries.
func TestDimmersSkipsDisabledLoadpoint(t *testing.T) {
	site := &Site{loadpoints: []*Loadpoint{nil}}
	assert.Empty(t, site.dimmers())
}
