package viessmann

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallationLabel(t *testing.T) {
	for _, tc := range []struct {
		inst  Installation
		label string
	}{
		{Installation{Description: "Home", Address: Address{Street: "Main St", City: "Berlin"}}, "Home"},
		{Installation{Description: "  ", Address: Address{City: "Berlin"}}, "Berlin"},
		{Installation{Address: Address{Street: "Main St", HouseNumber: "1", City: "Berlin"}}, "Main St 1, Berlin"},
		{Installation{Address: Address{Street: "Main St"}}, "Main St"},
		{Installation{Address: Address{City: "Berlin"}}, "Berlin"},
		{Installation{}, ""},
	} {
		assert.Equal(t, tc.label, tc.inst.label())
	}
}

func TestEquipmentOptions(t *testing.T) {
	installations := []Installation{
		{ID: 3242119, Description: "Home", Gateways: []Gateway{{Serial: "7472258009383263", GatewayType: "TCU301 "}, {Serial: "7472258009383262", GatewayType: "VitoconnectOPTO2"}}},
		{ID: 1000001, Address: Address{Street: "Main St", HouseNumber: "1", City: "Berlin"}, Gateways: []Gateway{{Serial: "7472258009383262", GatewayType: "VitoconnectOPTO2"}}},
		{ID: 2000002},
	}

	// all options match: single-item lists are auto-filled by the config UI
	assert.Equal(t, []service.Option{
		{Value: "1000001", Label: "Main St 1, Berlin", Hint: "7472258009383262", Match: true},
		{Value: "2000002", Match: true},
		{Value: "3242119", Label: "Home", Hint: "7472258009383263, 7472258009383262", Match: true},
	}, equipmentOptions(installations, false))

	// shared gateway listed once, attributed to the first installation
	assert.Equal(t, []service.Option{
		{Value: "7472258009383262", Label: "VitoconnectOPTO2", Hint: "Home", Match: true},
		{Value: "7472258009383263", Label: "TCU301", Hint: "Home", Match: true},
	}, equipmentOptions(installations, true))

	// empty list, not null
	assert.Equal(t, []service.Option{}, equipmentOptions(nil, false))
	assert.Equal(t, []service.Option{}, equipmentOptions(nil, true))
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
