package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deleteCircuitTestSite struct {
	site.API
	gridMeterRef string
	pvMeterRefs  []string
}

func (s deleteCircuitTestSite) GetGridMeterRef() string {
	return s.gridMeterRef
}

func (s deleteCircuitTestSite) GetPVMeterRefs() []string {
	return s.pvMeterRefs
}

func (s deleteCircuitTestSite) GetBatteryMeterRefs() []string {
	return nil
}

func (s deleteCircuitTestSite) GetAuxMeterRefs() []string {
	return nil
}

func (s deleteCircuitTestSite) GetExtMeterRefs() []string {
	return nil
}

func (s deleteCircuitTestSite) GetConsumerMeterRefs() []string {
	return nil
}

func TestDeleteCircuitMeter(t *testing.T) {
	tests := []struct {
		name         string
		gridMeterRef bool
		wantMeter    bool
	}{
		{
			name:      "dedicated meter",
			wantMeter: false,
		},
		{
			name:         "grid meter",
			gridMeterRef: true,
			wantMeter:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, db.NewInstance("sqlite", ":memory:"))
			config.Reset()

			meterConfig, err := config.AddConfig(templates.Meter, map[string]any{"type": "custom"})
			require.NoError(t, err)
			var meterInstance api.Meter
			require.NoError(t, config.Meters().Add(config.NewConfigurableDevice(&meterConfig, meterInstance)))

			circuitConfig, err := config.AddConfig(templates.Circuit, map[string]any{
				"meter": config.NameForID(meterConfig.ID),
			})
			require.NoError(t, err)
			var circuitInstance api.Circuit
			require.NoError(t, config.Circuits().Add(config.NewConfigurableDevice(&circuitConfig, circuitInstance)))

			gridMeterRef := ""
			if tt.gridMeterRef {
				gridMeterRef = config.NameForID(meterConfig.ID)
			}
			testSite := deleteCircuitTestSite{gridMeterRef: gridMeterRef}
			req := httptest.NewRequest(http.MethodDelete, "/", nil)
			req = mux.SetURLVars(req, map[string]string{"class": "circuit", "id": strconv.Itoa(circuitConfig.ID)})
			rec := httptest.NewRecorder()

			deleteDeviceHandler(testSite)(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			_, err = config.Circuits().ByName(config.NameForID(circuitConfig.ID))
			assert.Error(t, err)
			_, err = config.Meters().ByName(config.NameForID(meterConfig.ID))
			if tt.wantMeter {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestDeleteCircuitMeterSharedReference guards that a circuit's dedicated meter is not
// deleted out from under another reference to it - here, a second circuit still pointing
// at the same meter name.
func TestDeleteCircuitMeterSharedReference(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	config.Reset()

	meterConfig, err := config.AddConfig(templates.Meter, map[string]any{"type": "custom"})
	require.NoError(t, err)
	var meterInstance api.Meter
	require.NoError(t, config.Meters().Add(config.NewConfigurableDevice(&meterConfig, meterInstance)))

	meterName := config.NameForID(meterConfig.ID)

	circuit1Config, err := config.AddConfig(templates.Circuit, map[string]any{"meter": meterName})
	require.NoError(t, err)
	var circuit1Instance api.Circuit
	require.NoError(t, config.Circuits().Add(config.NewConfigurableDevice(&circuit1Config, circuit1Instance)))

	circuit2Config, err := config.AddConfig(templates.Circuit, map[string]any{"meter": meterName})
	require.NoError(t, err)
	var circuit2Instance api.Circuit
	require.NoError(t, config.Circuits().Add(config.NewConfigurableDevice(&circuit2Config, circuit2Instance)))

	testSite := deleteCircuitTestSite{}

	// deleting circuit1 must not take the meter down with it - circuit2 still uses it
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = mux.SetURLVars(req, map[string]string{"class": "circuit", "id": strconv.Itoa(circuit1Config.ID)})
	rec := httptest.NewRecorder()
	deleteDeviceHandler(testSite)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	_, err = config.Circuits().ByName(config.NameForID(circuit1Config.ID))
	assert.Error(t, err, "circuit1 should be deleted")
	_, err = config.Meters().ByName(meterName)
	assert.NoError(t, err, "shared meter must survive while circuit2 still references it")

	// deleting circuit2 (the last reference) may now take the meter with it
	req = httptest.NewRequest(http.MethodDelete, "/", nil)
	req = mux.SetURLVars(req, map[string]string{"class": "circuit", "id": strconv.Itoa(circuit2Config.ID)})
	rec = httptest.NewRecorder()
	deleteDeviceHandler(testSite)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	_, err = config.Meters().ByName(meterName)
	assert.Error(t, err, "meter should be deleted once its last reference is gone")
}
