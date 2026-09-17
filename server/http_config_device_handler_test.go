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
}

func (s deleteCircuitTestSite) GetGridMeterRef() string {
	return s.gridMeterRef
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
