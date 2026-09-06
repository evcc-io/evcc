package homeassistant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClimateAttributes(t *testing.T) {
	var state ClimateStateResponse
	err := json.Unmarshal([]byte(`{
		"entity_id": "climate.thermostat",
		"state": "heat",
		"attributes": {
			"temperature": 20.5,
			"current_temperature": 22.87,
			"min_temp": 5.0,
			"max_temp": 30.0,
			"target_temp_step": 0.1,
			"hvac_modes": ["heat"],
			"supported_features": 1
		}
	}`), &state)
	require.NoError(t, err)
	assert.Equal(t, "climate.thermostat", state.EntityId)
	assert.Equal(t, "heat", state.State)
	assert.Empty(t, state.Attributes.UnitOfMeasurement)
	assert.Equal(t, []string{"heat"}, state.Attributes.HVACModes)
	assert.Equal(t, 1, state.Attributes.SupportedFeatures)

	for _, tc := range []struct {
		got  *float64
		want float64
	}{
		{state.Attributes.Temperature, 20.5},
		{state.Attributes.CurrentTemperature, 22.87},
		{state.Attributes.MinTemp, 5},
		{state.Attributes.MaxTemp, 30},
		{state.Attributes.TargetTempStep, 0.1},
	} {
		require.NotNil(t, tc.got)
		assert.Equal(t, tc.want, *tc.got)
	}
}

func TestClimateAttributesMissingAndZero(t *testing.T) {
	for _, tc := range []struct {
		name       string
		attributes string
		missing    bool
	}{
		{"missing", `{}`, true},
		{"null", `{"temperature":null,"current_temperature":null,"min_temp":null,"max_temp":null,"target_temp_step":null,"hvac_modes":null,"supported_features":null}`, true},
		{"zero", `{"temperature":0,"current_temperature":0,"min_temp":0,"max_temp":0,"target_temp_step":0,"supported_features":0}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var state ClimateStateResponse
			require.NoError(t, json.Unmarshal([]byte(`{"attributes":`+tc.attributes+`}`), &state))
			assert.Nil(t, state.Attributes.HVACModes)
			assert.Zero(t, state.Attributes.SupportedFeatures)
			for _, got := range []*float64{
				state.Attributes.Temperature,
				state.Attributes.CurrentTemperature,
				state.Attributes.MinTemp,
				state.Attributes.MaxTemp,
				state.Attributes.TargetTempStep,
			} {
				if tc.missing {
					assert.Nil(t, got)
				} else {
					require.NotNil(t, got)
					assert.Zero(t, *got)
				}
			}
		})
	}
}

func TestClimateAttributesInvalid(t *testing.T) {
	for _, attributes := range []string{
		`"temperature":"20.5"`,
		`"current_temperature":"unknown"`,
		`"min_temp":false`,
		`"max_temp":{}`,
		`"target_temp_step":[]`,
		`"temperature":1e400`,
		`"temperature":NaN`,
		`"hvac_modes":[1]`,
		`"supported_features":"1"`,
		`"supported_features":1.5`,
	} {
		t.Run(attributes, func(t *testing.T) {
			var state ClimateStateResponse
			assert.Error(t, json.Unmarshal([]byte(`{"attributes":{`+attributes+`}}`), &state))
		})
	}
}

func TestStateResponseIgnoresClimateAttributes(t *testing.T) {
	var state StateResponse
	require.NoError(t, json.Unmarshal([]byte(`{
		"entity_id":"sensor.example",
		"state":"42",
		"attributes":{"temperature":"unknown","unit_of_measurement":"W"}
	}`), &state))
	assert.Equal(t, "42", state.State)
	assert.Equal(t, "W", state.Attributes.UnitOfMeasurement)
}

func TestGetClimateState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"heat", http.StatusOK, `{"state":"heat","attributes":{"temperature":20.5}}`, nil},
		{"unknown", http.StatusOK, `{"state":"unknown"}`, api.ErrNotAvailable},
		{"unavailable", http.StatusOK, `{"state":"unavailable"}`, api.ErrNotAvailable},
		{"http error", http.StatusServiceUnavailable, `unavailable`, nil},
		{"invalid temperature", http.StatusOK, `{"state":"heat","attributes":{"temperature":"unknown"}}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/states/climate.thermostat%2Fzone", r.URL.EscapedPath())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			srv.Start()

			state, err := newTestConnection(srv.URL).GetClimateState("climate.thermostat/zone")
			if tc.want != nil {
				assert.ErrorIs(t, err, tc.want)
			} else if tc.name != "heat" {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "heat", state.State)
				require.NotNil(t, state.Attributes.Temperature)
				assert.Equal(t, 20.5, *state.Attributes.Temperature)
			}
		})
	}
}
