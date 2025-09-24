package vehicle

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCapacityPlugin(t *testing.T) {
	tests := []struct {
		name           string
		staticCapacity float64
		pluginConfig   *plugin.Config
		pluginResult   float64
		pluginError    error
		expectedResult float64
	}{
		{
			name:           "static capacity only",
			staticCapacity: 75.0,
			pluginConfig:   nil,
			expectedResult: 75.0,
		},
		{
			name:           "plugin success overrides static",
			staticCapacity: 75.0,
			pluginConfig: &plugin.Config{
				Source: "const",
				Other:  map[string]any{"value": 85.0},
			},
			expectedResult: 85.0,
		},
		{
			name:           "plugin error falls back to static",
			staticCapacity: 75.0,
			pluginConfig: &plugin.Config{
				Source: "http",
				Other:  map[string]any{"uri": "http://invalid-url"},
			},
			expectedResult: 75.0, // Falls back to static on plugin error
		},
		{
			name:           "zero static capacity with plugin",
			staticCapacity: 0.0,
			pluginConfig: &plugin.Config{
				Source: "const",
				Other:  map[string]any{"value": 90.0},
			},
			expectedResult: 90.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embed := &embed{
				Capacity_: tt.staticCapacity,
			}

			// Initialize plugin if provided
			if tt.pluginConfig != nil {
				getter, err := tt.pluginConfig.FloatGetter(context.Background())
				if err == nil {
					embed.GetCapacity = getter
				}
				// If plugin initialization fails, GetCapacity remains nil
				// and we fall back to static capacity (which is the expected behavior)
			}

			result := embed.Capacity()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestVehicleConfigurableGetCapacity(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedError  string
		expectedResult float64
	}{
		{
			name: "configurable vehicle with capacity plugin",
			config: map[string]interface{}{
				"capacity": 75.0,
				"soc": map[string]interface{}{
					"source": "const",
					"value":  50.0,
				},
				"getCapacity": map[string]interface{}{
					"source": "const",
					"value":  85.0,
				},
			},
			expectedResult: 85.0,
		},
		{
			name: "configurable vehicle without capacity plugin",
			config: map[string]interface{}{
				"capacity": 80.0,
				"soc": map[string]interface{}{
					"source": "const",
					"value":  60.0,
				},
			},
			expectedResult: 80.0,
		},
		{
			name: "invalid capacity plugin config",
			config: map[string]interface{}{
				"capacity": 75.0,
				"soc": map[string]interface{}{
					"source": "const",
					"value":  50.0,
				},
				"getCapacity": map[string]interface{}{
					"source": "invalid-source",
				},
			},
			expectedError: "getCapacity:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vehicle, err := NewConfigurableFromConfig(ctx, tt.config)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, vehicle.Capacity())
		})
	}
}