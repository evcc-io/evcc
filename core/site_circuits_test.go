package core

import (
	"errors"
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
	dimCalls []float64
}

func (m *dimmableMeter) Dimmed() (bool, error) {
	m.gets++
	return m.dimmed, m.getErr
}

func (m *dimmableMeter) Dim(limit float64) error {
	m.dimCalls = append(m.dimCalls, limit)
	if m.dimErr != nil {
		return m.dimErr
	}
	m.dimmed = limit > 0
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

	require.NoError(t, site.dimMeters(4200))
	assert.Equal(t, []float64{4200}, m.dimCalls)
	assert.Equal(t, 1, m.gets)

	for range 3 {
		require.NoError(t, site.dimMeters(4200))
	}
	assert.Equal(t, []float64{4200}, m.dimCalls)
	assert.Equal(t, 1, m.gets)

	// changed limit is written
	require.NoError(t, site.dimMeters(3000))
	assert.Equal(t, []float64{4200, 3000}, m.dimCalls)

	require.NoError(t, site.dimMeters(0))
	assert.Equal(t, []float64{4200, 3000, 0}, m.dimCalls)

	// failed write is retried
	m.dimErr = errors.New("nope")
	require.Error(t, site.dimMeters(4200))
	require.Error(t, site.dimMeters(4200))
	assert.Equal(t, []float64{4200, 3000, 0, 4200, 4200}, m.dimCalls)
}

// A device that cannot report its state is written once, not on every cycle.
func TestDimMetersNotAvailable(t *testing.T) {
	m := &dimmableMeter{getErr: api.ErrNotAvailable}
	site := dimSite(m)

	for range 3 {
		require.NoError(t, site.dimMeters(4200))
	}
	assert.Equal(t, []float64{4200}, m.dimCalls)

	require.NoError(t, site.dimMeters(0))
	assert.Equal(t, []float64{4200, 0}, m.dimCalls)
}
