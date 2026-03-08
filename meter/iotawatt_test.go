package meter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// iotawattHandler serves mock IoTaWatt API responses
type iotawattHandler struct {
	series string
}

func (h *iotawattHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("show") == "series" {
		fmt.Fprint(w, h.series)
		return
	}

	sel := r.URL.Query().Get("select")
	switch sel {
	case "[GridNet.watts]":
		fmt.Fprint(w, `[[5000]]`)
	case "[Solar.watts]":
		fmt.Fprint(w, `[[2500]]`)
	case "[Grid_A.watts,Grid_B.watts,Grid_C.watts]":
		fmt.Fprint(w, `[[1700,1600,1700]]`)
	case "[Grid_A.amps,Grid_B.amps,Grid_C.amps]":
		fmt.Fprint(w, `[[7.3,6.9,7.3]]`)
	case "[Grid_A.volts,Grid_B.volts,Grid_C.volts]":
		fmt.Fprint(w, `[[233,231,232]]`)
	case "[GridNet.wh]", "[Solar.wh]":
		fmt.Fprint(w, `[[100]]`)
	case "[Grid_A.wh,Grid_B.wh,Grid_C.wh]":
		fmt.Fprint(w, `[[40,30,30]]`)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "unexpected select: %s", sel)
	}
}

func newIoTaWattTestServer() *httptest.Server {
	return httptest.NewServer(&iotawattHandler{
		series: `{"series":[
			{"name":"GridNet","unit":"Watts"},
			{"name":"Solar","unit":"Watts"},
			{"name":"Grid_A","unit":"Watts"},
			{"name":"Grid_B","unit":"Watts"},
			{"name":"Grid_C","unit":"Watts"},
			{"name":"Input_0","unit":"Volts"}
		]}`,
	})
}

func TestIoTaWattSingleChannel(t *testing.T) {
	srv := newIoTaWattTestServer()
	defer srv.Close()

	m, err := NewIoTaWatt(srv.URL, []string{"GridNet"}, 0)
	require.NoError(t, err)

	t.Run("CurrentPower", func(t *testing.T) {
		power, err := m.CurrentPower()
		require.NoError(t, err)
		assert.Equal(t, 5000.0, power)
	})

	t.Run("MeterEnergy", func(t *testing.T) {
		em, ok := m.(api.MeterEnergy)
		require.True(t, ok, "expected api.MeterEnergy")

		// first call seeds, returns 0
		energy, err := em.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, 0.0, energy)

		// second call accumulates
		energy, err = em.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, 0.1, energy) // 100 Wh = 0.1 kWh
	})

	t.Run("NoPhaseInterfaces", func(t *testing.T) {
		_, ok := m.(api.PhasePowers)
		assert.False(t, ok, "should not implement PhasePowers for single channel")

		_, ok = m.(api.PhaseCurrents)
		assert.False(t, ok, "should not implement PhaseCurrents for single channel")

		_, ok = m.(api.PhaseVoltages)
		assert.False(t, ok, "should not implement PhaseVoltages for single channel")
	})
}

func TestIoTaWattSolarSingleChannel(t *testing.T) {
	srv := newIoTaWattTestServer()
	defer srv.Close()

	m, err := NewIoTaWatt(srv.URL, []string{"Solar"}, 0)
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 2500.0, power)
}

func TestIoTaWattThreePhase(t *testing.T) {
	srv := newIoTaWattTestServer()
	defer srv.Close()

	channels := []string{"Grid_A", "Grid_B", "Grid_C"}
	m, err := NewIoTaWatt(srv.URL, channels, 0)
	require.NoError(t, err)

	t.Run("CurrentPower", func(t *testing.T) {
		power, err := m.CurrentPower()
		require.NoError(t, err)
		assert.Equal(t, 5000.0, power) // 1700+1600+1700
	})

	t.Run("PhasePowers", func(t *testing.T) {
		pp, ok := m.(api.PhasePowers)
		require.True(t, ok, "expected api.PhasePowers")

		p1, p2, p3, err := pp.Powers()
		require.NoError(t, err)
		assert.Equal(t, 1700.0, p1)
		assert.Equal(t, 1600.0, p2)
		assert.Equal(t, 1700.0, p3)
	})

	t.Run("PhaseCurrents", func(t *testing.T) {
		pc, ok := m.(api.PhaseCurrents)
		require.True(t, ok, "expected api.PhaseCurrents")

		i1, i2, i3, err := pc.Currents()
		require.NoError(t, err)
		assert.Equal(t, 7.3, i1)
		assert.Equal(t, 6.9, i2)
		assert.Equal(t, 7.3, i3)
	})

	t.Run("PhaseVoltages", func(t *testing.T) {
		pv, ok := m.(api.PhaseVoltages)
		require.True(t, ok, "expected api.PhaseVoltages")

		v1, v2, v3, err := pv.Voltages()
		require.NoError(t, err)
		assert.Equal(t, 233.0, v1)
		assert.Equal(t, 231.0, v2)
		assert.Equal(t, 232.0, v3)
	})

	t.Run("MeterEnergy", func(t *testing.T) {
		em, ok := m.(api.MeterEnergy)
		require.True(t, ok, "expected api.MeterEnergy")

		// first call seeds
		energy, err := em.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, 0.0, energy)

		// second call: 40+30+30 = 100 Wh = 0.1 kWh
		energy, err = em.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, 0.1, energy)
	})
}

func TestIoTaWattInvalidChannel(t *testing.T) {
	srv := newIoTaWattTestServer()
	defer srv.Close()

	_, err := NewIoTaWatt(srv.URL, []string{"NonExistent"}, 0)
	assert.ErrorContains(t, err, "unknown iotawatt series: NonExistent")
}

func TestIoTaWattVoltageChannelRejected(t *testing.T) {
	srv := newIoTaWattTestServer()
	defer srv.Close()

	_, err := NewIoTaWatt(srv.URL, []string{"Input_0"}, 0)
	assert.ErrorContains(t, err, "has unit")
}

func TestIoTaWattInvalidChannelCount(t *testing.T) {
	srv := newIoTaWattTestServer()
	defer srv.Close()

	_, err := NewIoTaWatt(srv.URL, []string{"Grid_A", "Grid_B"}, 0)
	assert.ErrorContains(t, err, "1 (single-phase) or 3 (three-phase)")
}

func TestIoTaWattMissingChannels(t *testing.T) {
	_, err := NewIoTaWattFromConfig(map[string]any{
		"uri": "http://localhost",
	})
	assert.ErrorContains(t, err, "missing channels")
}

func TestIoTaWattEmptyChannelsFiltered(t *testing.T) {
	srv := newIoTaWattTestServer()
	defer srv.Close()

	// simulates template with only L1 set, L2/L3 empty
	m, err := NewIoTaWattFromConfig(map[string]any{
		"uri":      srv.URL,
		"channels": []string{"GridNet", "", ""},
	})
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 5000.0, power)

	// should be single-phase (no phase interfaces)
	_, ok := m.(api.PhasePowers)
	assert.False(t, ok)
}
