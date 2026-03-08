package iotawatt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockIoTaWatt() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/config.txt":
			fmt.Fprint(w, `{
				"derive3ph": true,
				"inputs": [
					{"channel":0,"name":"Input_0","type":"VT","vphase":0},
					{"channel":1,"name":"Grid_A","type":"CT","vphase":0},
					{"channel":2,"name":"Grid_B","type":"CT","vphase":120},
					{"channel":3,"name":"Grid_C","type":"CT","vphase":240},
					{"channel":5,"name":"Solar_A","type":"CT","vphase":0},
					{"channel":7,"name":"Solar_B","type":"CT","vphase":120},
					{"channel":10,"name":"Solar_C","type":"CT","vphase":240},
					null,
					{"channel":12,"name":"Pool","type":"CT","vphase":240}
				],
				"outputs": [
					{"name":"GridNet","units":"Watts"},
					{"name":"Solar","units":"Watts"}
				]
			}`)
		case r.URL.Query().Get("show") == "series":
			fmt.Fprint(w, `{"series":[
				{"name":"Input_0","unit":"Volts"},
				{"name":"Grid_A","unit":"Watts"},
				{"name":"Grid_B","unit":"Watts"},
				{"name":"Grid_C","unit":"Watts"},
				{"name":"Solar_A","unit":"Watts"},
				{"name":"Solar_B","unit":"Watts"},
				{"name":"Solar_C","unit":"Watts"},
				{"name":"Pool","unit":"Watts"},
				{"name":"GridNet","unit":"Watts"},
				{"name":"Solar","unit":"Watts"},
				{"name":"Grid_A_Current","unit":"Amps"}
			]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestGetSeriesService(t *testing.T) {
	device := newMockIoTaWatt()
	defer device.Close()

	t.Run("FilterWatts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/series?uri="+device.URL, nil)
		w := httptest.NewRecorder()

		getSeries(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var result []string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		assert.Equal(t, []string{"Grid_A", "Grid_B", "Grid_C", "Solar_A", "Solar_B", "Solar_C", "Pool", "GridNet", "Solar"}, result)
	})

	t.Run("FilterAmps", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/series?uri="+device.URL+"&unit=Amps", nil)
		w := httptest.NewRecorder()

		getSeries(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var result []string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		assert.Equal(t, []string{"Grid_A_Current"}, result)
	})

	t.Run("MissingURI", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/series", nil)
		w := httptest.NewRecorder()

		getSeries(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetConfigService(t *testing.T) {
	device := newMockIoTaWatt()
	defer device.Close()

	t.Run("ThreePhase", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/config?uri="+device.URL, nil)
		w := httptest.NewRecorder()

		getConfig(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var result configResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))

		assert.True(t, result.ThreePhase)
		assert.Equal(t, 1, result.Phases["Grid_A"])
		assert.Equal(t, 2, result.Phases["Grid_B"])
		assert.Equal(t, 3, result.Phases["Grid_C"])
		assert.Equal(t, 1, result.Phases["Solar_A"])
		assert.Equal(t, 2, result.Phases["Solar_B"])
		assert.Equal(t, 3, result.Phases["Solar_C"])
		assert.Equal(t, 3, result.Phases["Pool"])
		// VT (Input_0) should not appear
		_, ok := result.Phases["Input_0"]
		assert.False(t, ok)
	})

	t.Run("SinglePhase", func(t *testing.T) {
		device := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{
				"derive3ph": false,
				"inputs": [
					{"channel":0,"name":"Voltage","type":"VT","vphase":0},
					{"channel":1,"name":"Grid","type":"CT","vphase":0},
					{"channel":2,"name":"Solar","type":"CT","vphase":0}
				]
			}`)
		}))
		defer device.Close()

		req := httptest.NewRequest("GET", "/config?uri="+device.URL, nil)
		w := httptest.NewRecorder()

		getConfig(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var result configResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))

		assert.False(t, result.ThreePhase)
		assert.Equal(t, 1, result.Phases["Grid"])
		assert.Equal(t, 1, result.Phases["Solar"])
	})

	t.Run("MissingURI", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()

		getConfig(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
