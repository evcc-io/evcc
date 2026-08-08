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
	setErr   error
	gets     int
	sets     int
	setCalls []int
}

func (m *curtailableMeter) CurtailedPercent() (int, error) {
	m.gets++
	return m.percent, nil
}

func (m *curtailableMeter) SetCurtailPercent(percent int) error {
	m.sets++
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

// dimmableMeter counts device interactions to verify caching.
type dimmableMeter struct {
	api.Meter
	dimmed   bool
	dimErr   error
	gets     int
	dimCalls []bool
}

func (m *dimmableMeter) Dimmed() (bool, error) {
	m.gets++
	return m.dimmed, nil
}

func (m *dimmableMeter) Dim(dim bool) error {
	m.dimCalls = append(m.dimCalls, dim)
	if m.dimErr != nil {
		return m.dimErr
	}
	m.dimmed = dim
	return nil
}

func TestDimMetersCache(t *testing.T) {
	m := &dimmableMeter{}
	site := &Site{
		log:       util.NewLogger("foo"),
		auxMeters: []config.Device[api.Meter]{config.NewStaticDevice(config.Named{}, api.Meter(m))},
	}

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
