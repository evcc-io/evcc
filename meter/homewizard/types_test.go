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

// Test homewizard kWh Meter 1-Phase response
func TestUnmarshalKwhDataResponse(t *testing.T) {
	{
		var res DataResponse
		// https://www.homewizard.com/shop/wi-fi-kwh-meter-1-phase/
		jsonstr := `{"wifi_ssid": "My Wi-Fi","wifi_strength": 100,"total_power_import_t1_kwh": 30.511,"total_power_export_t1_kwh": 85.951,"active_power_w": 543,"active_power_l1_w": 28,"active_power_l2_w": 0,"active_power_l3_w": -181,"active_voltage_l1_v": 235.4,"active_voltage_l2_v": 235.8,"active_voltage_l3_v": 236.1,"active_current_l1_a": 1.19,"active_current_l2_a": 0.37,"active_current_l3_a": -0.93}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, float64(30.511), res.TotalPowerImportT1kWh+res.TotalPowerImportT2kWh+res.TotalPowerImportT3kWh+res.TotalPowerImportT4kWh)
		assert.Equal(t, float64(85.951), res.TotalPowerExportT1kWh+res.TotalPowerExportT2kWh+res.TotalPowerExportT3kWh+res.TotalPowerExportT4kWh)
		assert.Equal(t, float64(543), res.ActivePowerW)
		assert.Equal(t, float64(235.4), res.ActiveVoltageL1V)
		assert.Equal(t, float64(235.8), res.ActiveVoltageL2V)
		assert.Equal(t, float64(236.1), res.ActiveVoltageL3V)
		assert.Equal(t, float64(1.19), res.ActiveCurrentL1A)
		assert.Equal(t, float64(0.37), res.ActiveCurrentL2A)
		assert.Equal(t, float64(-0.93), res.ActiveCurrentL3A)
	}
}

// Test homewizard P1 Meter response
func TestUnmarshalP1DataResponse(t *testing.T) {
	{
		var res DataResponse
		// https://www.homewizard.com/shop/wi-fi-p1-meter-rj12-2/
		jsonstr := `{"wifi_ssid":"redacted","wifi_strength":78,"smr_version":50,"meter_model":"Landis + Gyr","unique_id":"redacted","active_tariff":2,"total_power_import_kwh":18664.997,"total_power_import_t1_kwh":10909.724,"total_power_import_t2_kwh":7755.273,"total_power_export_kwh":13823.608,"total_power_export_t1_kwh":4243.981,"total_power_export_t2_kwh":9579.627,"active_power_w":203.000,"active_power_l1_w":-21.000,"active_power_l2_w":57.000,"active_power_l3_w":168.000,"active_voltage_l1_v":228.000,"active_voltage_l2_v":226.000,"active_voltage_l3_v":225.000,"active_current_a":1.091,"active_current_l1_a":-0.092,"active_current_l2_a":0.252,"active_current_l3_a":0.747,"voltage_sag_l1_count":12.000,"voltage_sag_l2_count":12.000,"voltage_sag_l3_count":19.000,"voltage_swell_l1_count":5055.000,"voltage_swell_l2_count":1950.000,"voltage_swell_l3_count":0.000,"any_power_fail_count":12.000,"long_power_fail_count":2.000,"total_gas_m3":5175.363,"gas_timestamp":241106093006,"gas_unique_id":"redacted","external":[{"unique_id":"redacted","type":"gas_meter","timestamp":241106093006,"value":5175.363,"unit":"m3"}]}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, float64(18664.997), res.TotalPowerImportT1kWh+res.TotalPowerImportT2kWh+res.TotalPowerImportT3kWh+res.TotalPowerImportT4kWh)
		assert.Equal(t, float64(13823.608), res.TotalPowerExportT1kWh+res.TotalPowerExportT2kWh+res.TotalPowerExportT3kWh+res.TotalPowerExportT4kWh)
		assert.Equal(t, float64(203), res.ActivePowerW)
		assert.Equal(t, float64(228), res.ActiveVoltageL1V)
		assert.Equal(t, float64(226), res.ActiveVoltageL2V)
		assert.Equal(t, float64(225), res.ActiveVoltageL3V)
		assert.Equal(t, float64(-0.092), res.ActiveCurrentL1A)
		assert.Equal(t, float64(0.252), res.ActiveCurrentL2A)
		assert.Equal(t, float64(0.747), res.ActiveCurrentL3A)
	}
}
