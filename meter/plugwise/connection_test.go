package plugwise

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnection(t *testing.T) {
	c, err := NewConnection("192.168.0.1", "testpass", time.Second)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestCurrentPower(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	power, err := c.CurrentPower()
	require.NoError(t, err)
	// electricity_consumed (312.0 W) - electricity_produced (0.0 W) = 312.0 W
	assert.Equal(t, 312.0, power)
}

func TestCaching(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	// First call — hits the server
	_, err = c.CurrentPower()
	require.NoError(t, err)

	// Second call within the 1-second TTL — must be served from cache
	_, err = c.CurrentPower()
	require.NoError(t, err)

	// Only one HTTP request should have been made
	assert.Equal(t, int32(1), requestCount.Load(), "second CurrentPower() call within TTL must not make a second HTTP request")
}

func TestPhasePowers(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	// Fixture: phase_{one,two,three}_consumed = 31/203/77 W; all produced = 0.
	// Net = consumed - produced per phase (D-02).
	p1, p2, p3, err := c.PhasePowers()
	require.NoError(t, err)
	assert.Equal(t, 31.0, p1)
	assert.Equal(t, 203.0, p2)
	assert.Equal(t, 77.0, p3)
}

func TestPhaseVoltages(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	// Fixture: voltage_phase_{one,two,three} = 232.3/229.8/231.2 V.
	v1, v2, v3, err := c.PhaseVoltages()
	require.NoError(t, err)
	assert.Equal(t, 232.3, v1)
	assert.Equal(t, 229.8, v2)
	assert.Equal(t, 231.2, v3)
}

func TestCurrents(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	// Derived: I_n = P_n / V_n per phase.
	i1, i2, i3, err := c.Currents()
	require.NoError(t, err)
	assert.InDelta(t, 31.0/232.3, i1, 1e-6)
	assert.InDelta(t, 203.0/229.8, i2, 1e-6)
	assert.InDelta(t, 77.0/231.2, i3, 1e-6)
}

func TestCurrentsZeroVoltage(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	// Inline-mutate the raw XML bytes: zero out voltage_phase_one (232.30 -> 0.00).
	// Keeps testdata/ minimal; self-documenting per research open question #1.
	zeroFixture := bytes.ReplaceAll(fixture, []byte(">232.30<"), []byte(">0.00<"))
	require.NotEqual(t, len(fixture), 0)
	require.Contains(t, string(zeroFixture), ">0.00<", "substitution must have occurred")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zeroFixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	i1, i2, i3, err := c.Currents()
	assert.Equal(t, 0.0, i1)
	assert.Equal(t, 0.0, i2)
	assert.Equal(t, 0.0, i3)
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}
