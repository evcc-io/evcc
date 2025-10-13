package cardata

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestStatus_PHEV_Fallback(t *testing.T) {
	tests := []struct {
		name              string
		hvStatus          string
		hvStatusErr       bool
		chargingStatus    string
		chargingStatusErr bool
		portStatus        string
		expected          api.ChargeStatus
	}{
		{
			name:           "BEV charging",
			hvStatus:       "CHARGING",
			chargingStatus: "CHARGINGACTIVE",
			portStatus:     "CONNECTED",
			expected:       api.StatusC,
		},
		{
			name:           "PHEV charging with invalid hvStatus",
			hvStatus:       "INVALID",
			chargingStatus: "CHARGINGACTIVE",
			portStatus:     "CONNECTED",
			expected:       api.StatusC,
		},
		{
			name:            "PHEV charging with unavailable hvStatus",
			hvStatusErr:     true,
			chargingStatus:  "CHARGINGACTIVE",
			portStatus:      "CONNECTED",
			expected:        api.StatusC,
		},
		{
			name:           "PHEV not charging",
			hvStatus:       "INVALID",
			chargingStatus: "NOCHARGING",
			portStatus:     "CONNECTED",
			expected:       api.StatusB,
		},
		{
			name:           "Disconnected",
			hvStatus:       "INVALID",
			chargingStatus: "NOCHARGING",
			portStatus:     "DISCONNECTED",
			expected:       api.StatusA,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := &Provider{
				streaming: make(map[string]StreamingData),
				initial:   make(map[string]TelematicDataPoint),
			}

			// Set up test data
			v.initial["vehicle.body.chargingPort.status"] = TelematicDataPoint{Value: tc.portStatus}
			
			if !tc.hvStatusErr {
				v.initial["vehicle.drivetrain.electricEngine.charging.hvStatus"] = TelematicDataPoint{Value: tc.hvStatus}
			}
			
			if !tc.chargingStatusErr {
				v.initial["vehicle.drivetrain.electricEngine.charging.status"] = TelematicDataPoint{Value: tc.chargingStatus}
			}

			status, err := v.Status()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, status)
		})
	}
}

func TestClimater_PHEV_Fallback(t *testing.T) {
	tests := []struct {
		name                 string
		comfortState         string
		comfortStateErr      bool
		preconditionActivity string
		expected             bool
	}{
		{
			name:                 "BEV heating",
			comfortState:         "COMFORT_HEATING",
			preconditionActivity: "HEATING",
			expected:             true,
		},
		{
			name:                 "PHEV heating with empty comfortState",
			comfortState:         "",
			preconditionActivity: "HEATING",
			expected:             true,
		},
		{
			name:                 "PHEV cooling",
			comfortStateErr:      true,
			preconditionActivity: "COOLING",
			expected:             true,
		},
		{
			name:                 "PHEV ventilation",
			comfortStateErr:      true,
			preconditionActivity: "VENTILATION",
			expected:             true,
		},
		{
			name:                 "PHEV inactive",
			comfortStateErr:      true,
			preconditionActivity: "INACTIVE",
			expected:             false,
		},
		{
			name:                 "PHEV standby",
			comfortState:         "",
			preconditionActivity: "STANDBY",
			expected:             false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := &Provider{
				streaming: make(map[string]StreamingData),
				initial:   make(map[string]TelematicDataPoint),
			}

			// Set up test data
			if !tc.comfortStateErr {
				v.initial["vehicle.cabin.hvac.preconditioning.status.comfortState"] = TelematicDataPoint{Value: tc.comfortState}
			}
			
			v.initial["vehicle.vehicle.preConditioning.activity"] = TelematicDataPoint{Value: tc.preconditionActivity}

			active, err := v.Climater()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, active)
		})
	}
}
