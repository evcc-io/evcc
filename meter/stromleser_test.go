package meter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStromleserServer(t *testing.T, resp map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestStromleserGridPower(t *testing.T) {
	ts := newStromleserServer(t, map[string]string{
		"device_id": "STROM_ONE_6029750131",
		"16.7.0":    "4 W",
		"1.8.0":     "35.789 kWh",
		"2.8.0":     "19929.796 kWh",
	})
	defer ts.Close()

	m, err := NewStromleserFromConfig(map[string]any{"uri": ts.URL, "usage": "grid"})
	require.NoError(t, err)

	power, err := m.(interface{ CurrentPower() (float64, error) }).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 4.0, power)
}

func TestStromleserGridEnergy(t *testing.T) {
	ts := newStromleserServer(t, map[string]string{
		"1.8.0":  "365.084 kWh",
		"2.8.0":  "41037.214 kWh",
		"16.7.0": "0 W",
	})
	defer ts.Close()

	m, err := NewStromleserFromConfig(map[string]any{"uri": ts.URL, "usage": "grid"})
	require.NoError(t, err)

	energy, err := m.(interface{ TotalEnergy() (float64, error) }).TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 365.084, energy)
}

func TestStromleserPVPower(t *testing.T) {
	ts := newStromleserServer(t, map[string]string{
		"16.7.0": "4 W",
		"1.8.0":  "35.789 kWh",
		"2.8.0":  "19929.796 kWh",
	})
	defer ts.Close()

	m, err := NewStromleserFromConfig(map[string]any{"uri": ts.URL, "usage": "pv"})
	require.NoError(t, err)

	power, err := m.(interface{ CurrentPower() (float64, error) }).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, -4.0, power)
}

func TestStromleserPVEnergy(t *testing.T) {
	ts := newStromleserServer(t, map[string]string{
		"16.7.0": "-500 W",
		"1.8.0":  "35.789 kWh",
		"2.8.0":  "19929.796 kWh",
	})
	defer ts.Close()

	m, err := NewStromleserFromConfig(map[string]any{"uri": ts.URL, "usage": "pv"})
	require.NoError(t, err)

	energy, err := m.(interface{ TotalEnergy() (float64, error) }).TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 19929.796, energy)
}

func TestStromleserFallbackPower(t *testing.T) {
	// device returns 1.7.0 / 2.7.0 but not 16.7.0
	ts := newStromleserServer(t, map[string]string{
		"1.7.0": "100 W",
		"2.7.0": "0 W",
		"1.8.0": "365.084 kWh",
		"2.8.0": "41037.214 kWh",
	})
	defer ts.Close()

	m, err := NewStromleserFromConfig(map[string]any{"uri": ts.URL, "usage": "grid"})
	require.NoError(t, err)

	power, err := m.(interface{ CurrentPower() (float64, error) }).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 100.0, power)
}

func TestParseOBIS(t *testing.T) {
	cases := []struct {
		input    string
		expected float64
	}{
		{"4 W", 4.0},
		{"35.789 kWh", 35.789},
		{"-500 W", -500.0},
		{"0.000 W", 0.0},
		{"41037.214 kWh", 41037.214},
	}

	for _, tc := range cases {
		v, err := parseOBIS(tc.input)
		require.NoError(t, err, "input: %q", tc.input)
		assert.Equal(t, tc.expected, v, "input: %q", tc.input)
	}
}
