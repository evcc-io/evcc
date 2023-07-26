package homewizard

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test ApiResponse
func TestUnmarshalApiResponse(t *testing.T) {
	{
		var res ApiResponse

		jsonstr := `{"product_type": "HWE-SKT","product_name": "P1 Meter","serial": "3c39e7aabbcc","firmware_version": "2.11","api_version": "v1"}`
		assert.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, "HWE-SKT", res.ProductType)
		assert.Equal(t, "v1", res.ApiVersion)
	}
}

// Test StateResponse
func TestUnmarshalStateResponse(t *testing.T) {
	{
		var res StateResponse

		jsonstr := `{"power_on": true,"switch_lock": false,"brightness": 255}`
		assert.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, true, res.PowerOn)
	}
}

// Test DataResponse
func TestUnmarshalDataResponse(t *testing.T) {
	{
		var res DataResponse

		jsonstr := `{"wifi_ssid": "My Wi-Fi","wifi_strength": 100,"total_power_import_t1_kwh": 30.511,"total_power_export_t1_kwh": 85.951,"active_power_w": 543,"active_power_l1_w": 676}`
		assert.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, float64(30.511), res.TotalPowerImportT1kWh+res.TotalPowerImportT2kWh+res.TotalPowerImportT3kWh+res.TotalPowerImportT4kWh)
		assert.Equal(t, float64(543), res.ActivePowerW)
	}
}
