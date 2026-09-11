package viessmann

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValues(t *testing.T) {
	installations := []Installation{
		{ID: 3242119, Gateways: []Gateway{{Serial: "7472258009383263"}, {Serial: "7472258009383262"}}},
		{ID: 1000001, Gateways: []Gateway{{Serial: "7472258009383262"}}},
	}

	assert.Equal(t, []string{"1000001", "3242119"}, values(installations, false))
	assert.Equal(t, []string{"7472258009383262", "7472258009383263"}, values(installations, true))

	// empty list, not null
	assert.Equal(t, []string{}, values(nil, false))
	assert.Equal(t, []string{}, values(nil, true))
}

// The service handler must return an empty JSON list when the user has not
// authorized yet, as the config UI polls it while the form is being filled.
func TestServiceHandlerUnauthorized(t *testing.T) {
	for _, query := range []string{"", "&detail=gateways"} {
		req := httptest.NewRequest(http.MethodGet, "/equipment?clientid=&redirecturi="+query, nil)
		w := httptest.NewRecorder()
		serviceMux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, query)
		assert.JSONEq(t, "[]", w.Body.String(), query)
	}
}

func TestHasMeasurements(t *testing.T) {
	all := []Feature{
		{Feature: "heating.dhw.oneTimeCharge"},
		{Feature: "heating.power.consumption.current"},
		{Feature: "heating.dhw.sensors.temperature.dhwCylinder"},
		{Feature: "heating.dhw.temperature.main"},
	}
	assert.True(t, hasMeasurements(all))

	// any required data point missing -> not supported
	assert.False(t, hasMeasurements(all[:3]))
	assert.False(t, hasMeasurements(nil))
}

// Like /equipment, /measurements must return an empty JSON list until the user
// has authorized, as the config UI polls it while the form is being filled.
func TestMeasurementsHandlerUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/measurements?clientid=&redirecturi=&gateway_serial=&installation_id=&device_id=0", nil)
	w := httptest.NewRecorder()
	serviceMux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())
}
