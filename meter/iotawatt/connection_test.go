package iotawatt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowSeries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/query", r.URL.Path)
		assert.Equal(t, "series", r.URL.Query().Get("show"))

		fmt.Fprint(w, `{"series":[
			{"name":"Input_0","unit":"Volts"},
			{"name":"Grid_A","unit":"Watts"},
			{"name":"Solar","unit":"Watts"},
			{"name":"GridNet","unit":"Watts"}
		]}`)
	}))
	defer srv.Close()

	conn, err := NewConnection(srv.URL, time.Second)
	require.NoError(t, err)

	series, err := conn.ShowSeries()
	require.NoError(t, err)

	assert.Len(t, series, 4)
	assert.Equal(t, "Input_0", series[0].Name)
	assert.Equal(t, "Volts", series[0].Unit)
	assert.Equal(t, "GridNet", series[3].Name)
	assert.Equal(t, "Watts", series[3].Unit)
}

func TestValidateSeries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"series":[
			{"name":"Input_0","unit":"Volts"},
			{"name":"Grid_A","unit":"Watts"},
			{"name":"Grid_B","unit":"Watts"},
			{"name":"Grid_C","unit":"Watts"},
			{"name":"Grid_A_Current","unit":"Amps"},
			{"name":"Grid_B_Current","unit":"Amps"},
			{"name":"Grid_C_Current","unit":"Amps"},
			{"name":"GridNet","unit":"Watts"},
			{"name":"Solar","unit":"Watts"}
		]}`)
	}))
	defer srv.Close()

	conn, err := NewConnection(srv.URL, time.Second)
	require.NoError(t, err)

	// valid Watts series
	unit, err := conn.ValidateSeries([]string{"GridNet"}, "Watts")
	assert.NoError(t, err)
	assert.Equal(t, "Watts", unit)

	// valid Watts phases
	unit, err = conn.ValidateSeries([]string{"Grid_A", "Grid_B", "Grid_C"}, "Watts", "Amps")
	assert.NoError(t, err)
	assert.Equal(t, "Watts", unit)

	// valid Amps phases
	unit, err = conn.ValidateSeries([]string{"Grid_A_Current", "Grid_B_Current", "Grid_C_Current"}, "Watts", "Amps")
	assert.NoError(t, err)
	assert.Equal(t, "Amps", unit)

	// unknown series
	_, err = conn.ValidateSeries([]string{"NonExistent"}, "Watts")
	assert.ErrorContains(t, err, "unknown iotawatt series: NonExistent")

	// wrong unit
	_, err = conn.ValidateSeries([]string{"Input_0"}, "Watts")
	assert.ErrorContains(t, err, "has unit")

	// mixed units
	_, err = conn.ValidateSeries([]string{"Grid_A", "Grid_A_Current", "Grid_C"}, "Watts", "Amps")
	assert.ErrorContains(t, err, "mixed units")

	// empty list
	unit, err = conn.ValidateSeries(nil, "Watts")
	assert.NoError(t, err)
	assert.Equal(t, "", unit)
}

func TestQueryPower(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/query", r.URL.Path)

		sel := r.URL.Query().Get("select")
		assert.Equal(t, "s-10s", r.URL.Query().Get("begin"))
		assert.Equal(t, "s", r.URL.Query().Get("end"))
		assert.Equal(t, "all", r.URL.Query().Get("group"))

		switch sel {
		case "[GridNet.watts]":
			fmt.Fprint(w, `[[16554.17]]`)
		case "[Grid_A.watts,Grid_B.watts,Grid_C.watts]":
			fmt.Fprint(w, `[[5370,4992,5935]]`)
		default:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "unexpected select: %s", sel)
		}
	}))
	defer srv.Close()

	conn, err := NewConnection(srv.URL, time.Second)
	require.NoError(t, err)

	t.Run("single channel", func(t *testing.T) {
		values, err := conn.QueryPower("GridNet")
		require.NoError(t, err)
		assert.Len(t, values, 1)
		assert.InDelta(t, 16554.17, values[0], 0.01)
	})

	t.Run("multiple channels", func(t *testing.T) {
		values, err := conn.QueryPower("Grid_A", "Grid_B", "Grid_C")
		require.NoError(t, err)
		assert.Len(t, values, 3)
		assert.Equal(t, 5370.0, values[0])
		assert.Equal(t, 4992.0, values[1])
		assert.Equal(t, 5935.0, values[2])
	})
}

func TestQueryCurrents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sel := r.URL.Query().Get("select")
		assert.Equal(t, "[Grid_A.amps,Grid_B.amps,Grid_C.amps]", sel)
		fmt.Fprint(w, `[[19.2,17.2,22.6]]`)
	}))
	defer srv.Close()

	conn, err := NewConnection(srv.URL, time.Second)
	require.NoError(t, err)

	values, err := conn.QueryCurrents("Grid_A", "Grid_B", "Grid_C")
	require.NoError(t, err)
	assert.Equal(t, []float64{19.2, 17.2, 22.6}, values)
}

func TestQueryPowerNoChannels(t *testing.T) {
	conn, err := NewConnection("http://localhost", time.Second)
	require.NoError(t, err)

	_, err = conn.QueryPower()
	assert.ErrorContains(t, err, "no channels specified")
}

func TestQueryPowerUnexpectedResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{"empty response", `[]`},
		{"wrong column count", `[[1,2]]`},
		{"multiple rows", `[[1],[2]]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tc.response)
			}))
			defer srv.Close()

			conn, err := NewConnection(srv.URL, time.Second)
			require.NoError(t, err)

			_, err = conn.QueryPower("GridNet")
			assert.ErrorContains(t, err, "unexpected query response")
		})
	}
}

func TestTotalEnergy(t *testing.T) {
	var callCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		sel := r.URL.Query().Get("select")
		assert.Equal(t, "[Solar.wh]", sel)
		assert.Equal(t, "s", r.URL.Query().Get("end"))
		assert.Equal(t, "all", r.URL.Query().Get("group"))

		// begin should be a unix timestamp (not empty)
		begin := r.URL.Query().Get("begin")
		assert.NotEmpty(t, begin)

		// return 500 Wh for each call
		fmt.Fprint(w, `[[500]]`)
	}))
	defer srv.Close()

	conn, err := NewConnection(srv.URL, time.Second)
	require.NoError(t, err)

	// first call seeds the timestamp, returns 0
	energy, err := conn.TotalEnergy("Solar")
	require.NoError(t, err)
	assert.Equal(t, 0.0, energy)
	assert.Equal(t, 0, callCount)

	// second call queries and accumulates
	energy, err = conn.TotalEnergy("Solar")
	require.NoError(t, err)
	assert.Equal(t, 0.5, energy) // 500 Wh = 0.5 kWh
	assert.Equal(t, 1, callCount)

	// third call accumulates further
	energy, err = conn.TotalEnergy("Solar")
	require.NoError(t, err)
	assert.Equal(t, 1.0, energy) // 500 + 500 Wh = 1.0 kWh
	assert.Equal(t, 2, callCount)
}

func TestTotalEnergyOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	conn, err := NewConnection(srv.URL, time.Second)
	require.NoError(t, err)

	// seed
	_, err = conn.TotalEnergy("Solar")
	require.NoError(t, err)

	// on error, returns last known total
	energy, err := conn.TotalEnergy("Solar")
	assert.Error(t, err)
	assert.Equal(t, 0.0, energy)
}

func TestNewConnectionMissingURI(t *testing.T) {
	_, err := NewConnection("", time.Second)
	assert.ErrorContains(t, err, "missing uri")
}

func TestNewConnectionDefaultScheme(t *testing.T) {
	// should not panic with a hostname-only URI
	conn, err := NewConnection("iotawatt.local", time.Second)
	require.NoError(t, err)
	assert.Contains(t, conn.uri, "http://")
}

func TestNewConnectionTrailingSlash(t *testing.T) {
	conn, err := NewConnection("http://iotawatt.local/", time.Second)
	require.NoError(t, err)
	assert.Equal(t, "http://iotawatt.local", conn.uri)
}

func TestUnmarshalShowSeriesResponse(t *testing.T) {
	jsonstr := `{"series":[{"name":"voltage","unit":"Volts"},{"name":"mains1","unit":"Watts"},{"name":"solar","unit":"Watts"}]}`

	var res ShowSeriesResponse
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

	assert.Len(t, res.Series, 3)
	assert.Equal(t, "voltage", res.Series[0].Name)
	assert.Equal(t, "Volts", res.Series[0].Unit)
	assert.Equal(t, "mains1", res.Series[1].Name)
	assert.Equal(t, "Watts", res.Series[1].Unit)
}
