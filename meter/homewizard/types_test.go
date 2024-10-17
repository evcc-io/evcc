package homewizard

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ApiResponse
func TestUnmarshalApiResponse(t *testing.T) {
	{
		var res ApiResponse

		jsonstr := `{"product_type": "HWE-SKT","product_name": "P1 Meter","serial": "3c39e7aabbcc","firmware_version": "2.11","api_version": "v1"}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, "HWE-SKT", res.ProductType)
		assert.Equal(t, "v1", res.ApiVersion)
	}
}

// Test StateResponse
func TestUnmarshalStateResponse(t *testing.T) {
	{
		var res StateResponse

		jsonstr := `{"power_on": true,"switch_lock": false,"brightness": 255}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.True(t, res.PowerOn)
	}
}

// Test DataResponse
func TestUnmarshalDataResponse(t *testing.T) {
	{
		var res DataResponse

		jsonstr := `{"wifi_ssid": "My Wi-Fi","wifi_strength": 100,"total_power_import_t1_kwh": 30.511,"total_power_export_t1_kwh": 85.951,"active_power_w": 543,"active_power_l1_w": 28,"active_power_l2_w": 0,"active_power_l3_w": -181,"active_voltage_l1_v": 235.4,"active_voltage_l2_v": 235.8,"active_voltage_l3_v": 236.1,"active_current_l1_a": 1.19,"active_current_l2_a": 0.37,"active_current_l3_a": -0.93}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, float64(30.511), res.TotalPowerImportT1kWh+res.TotalPowerImportT2kWh+res.TotalPowerImportT3kWh+res.TotalPowerImportT4kWh)
		assert.Equal(t, float64(543), res.ActivePowerW)
		assert.Equal(t, float64(235.4), res.ActiveVoltageL1V)
		assert.Equal(t, float64(235.8), res.ActiveVoltageL2V)
		assert.Equal(t, float64(236.1), res.ActiveVoltageL3V)
		assert.Equal(t, float64(1.19), res.ActiveCurrentL1A)
		assert.Equal(t, float64(0.37), res.ActiveCurrentL2A)
		assert.Equal(t, float64(-0.93), res.ActiveCurrentL3A)
	}
}
