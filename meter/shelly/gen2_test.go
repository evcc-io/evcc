package shelly

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Gen2+ status responses
func TestUnmarshalGen2StatusResponse(t *testing.T) {
	{
		// Shelly Pro 1PM channel 0 (1)
		var res Gen2StatusResponse

		jsonstr := `{"ble":{},"cloud":{"connected":true},"eth":{"ip":null},"input:0":{"id":0,"state":false},"input:1":{"id":1,"state":false},"mqtt":{"connected":false},"switch:0":{"id":0, "source":"HTTP", "output":false, "apower":47.11, "voltage":232.0, "current":0.000, "pf":0.00, "aenergy":{"total":5.125,"by_minute":[0.000,0.000,0.000],"minute_ts":1675718520},"temperature":{"tC":25.3, "tF":77.5}},"sys":{"mac":"30C6F78BB4D8","restart_required":false,"time":"22:22","unixtime":1675718522,"uptime":45070,"ram_size":234204,"ram_free":137716,"fs_size":524288,"fs_free":172032,"cfg_rev":13,"kvs_rev":1,"schedule_rev":0,"webhook_rev":0,"available_updates":{"beta":{"version":"0.13.0-beta3"}}},"wifi":{"sta_ip":"192.168.178.64","status":"got ip","ssid":"***","rssi":-62},"ws":{"connected":false}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, 5.125, res.Switch0.Aenergy.Total)
		assert.Equal(t, 47.11, res.Switch0.Apower)
	}

	{
		// Shelly PM Mini Gen3 channel 0 (1)
		var res Gen2StatusResponse

		jsonstr := `{"ble":{},"cloud":{"connected":true},"mqtt":{"connected":false},"pm1:0":{"id":0, "voltage":239.9, "current":7.434, "apower":1780.1 ,"freq":50.1,"aenergy":{"total":3551.682,"by_minute":[15234.772,29611.247,29825.821],"minute_ts":1719917850},"ret_aenergy":{"total":0.000,"by_minute":[0.000,0.000,0.000],"minute_ts":1719917850}},"sys":{"mac":"84FCE638D818","restart_required":false,"time":"12:57","unixtime":1719917851,"uptime":62328,"ram_size":261744,"ram_free":151436,"fs_size":1048576,"fs_free":712704,"cfg_rev":10,"kvs_rev":1,"schedule_rev":0,"webhook_rev":0,"available_updates":{"stable":{"version":"1.3.3"}},"reset_reason":1},"wifi":{"sta_ip":"192.168.178.89","status":"got ip","ssid":"FritzBox 8 2.4","rssi":-62},"ws":{"connected":false}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, 3551.682, res.Pm0.Aenergy.Total)
		assert.Equal(t, 1780.1, res.Pm0.Apower)
	}

	{
		// Shelly Pro 3EM
		var res Gen2StatusResponse

		jsonstr := `{"ble":{},"bthome":{"errors":["bluetooth_disabled"]},"cloud":{"connected":true},"em1:0":{"id":0,"current":3.705,"voltage":242.8,"act_power":598.9,"aprt_power":900.6,"pf":0.66,"freq":50.0,"calibration":"factory"},"em1:1":{"id":1,"current":0.194,"voltage":242.8,"act_power":0.0,"aprt_power":47.2,"pf":0.00,"freq":50.0,"calibration":"factory"},"em1:2":{"id":2,"current":0.027,"voltage":242.8,"act_power":0.0,"aprt_power":6.6,"pf":0.00,"freq":50.0,"calibration":"factory"},"em1data:0":{"id":0,"total_act_energy":3458.24,"total_act_ret_energy":1605.24},"em1data:1":{"id":1,"total_act_energy":2768.67,"total_act_ret_energy":25.49},"em1data:2":{"id":2,"total_act_energy":3.09,"total_act_ret_energy":0.71},"eth":{"ip":null},"modbus":{},"mqtt":{"connected":false},"sys":{"mac":"FCE8C0DBA850","restart_required":false,"time":"19:46","unixtime":1731404780,"uptime":563,"ram_size":247148,"ram_free":110596,"fs_size":524288,"fs_free":176128,"cfg_rev":21,"kvs_rev":0,"schedule_rev":3,"webhook_rev":1,"available_updates":{},"reset_reason":3},"temperature:0":{"id": 0,"tC":39.0, "tF":102.2},"wifi":{"sta_ip":"192.168.40.174","status":"got ip","ssid":"IoT","rssi":-67},"ws":{"connected":false}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		// Channel 0 (1)
		assert.Equal(t, 598.9, res.Em0.ActPower)
		assert.Equal(t, 3.705, res.Em0.Current)
		assert.Equal(t, 242.8, res.Em0.Voltage)
		assert.Equal(t, 3458.24, res.Em0Data.TotalActEnergy)
		assert.Equal(t, 1605.24, res.Em0Data.TotalActRetEnergy)
		// Channel 1 (2)
		assert.Equal(t, 0.0, res.Em1.ActPower)
		assert.Equal(t, 0.194, res.Em1.Current)
		assert.Equal(t, 242.8, res.Em1.Voltage)
		assert.Equal(t, 2768.67, res.Em1Data.TotalActEnergy)
		assert.Equal(t, 25.49, res.Em1Data.TotalActRetEnergy)
		// Channel 2 (3)
		assert.Equal(t, 0.0, res.Em2.ActPower)
		assert.Equal(t, 0.027, res.Em2.Current)
		assert.Equal(t, 242.8, res.Em2.Voltage)
		assert.Equal(t, 3.09, res.Em2Data.TotalActEnergy)
		assert.Equal(t, 0.71, res.Em2Data.TotalActRetEnergy)
	}

	{
		// Shelly Pro EM-50 channel 0 + 1
		var res Gen2StatusResponse

		jsonstr := `{"ble":{},"bthome":{"errors":["bluetooth_disabled"]},"cloud":{"connected":true},"em1:0":{"id":0,"current":1.473,"voltage":226.9,"act_power":-332.2,"aprt_power":335.0,"pf":0.99,"freq":50.0,"calibration":"factory"},"em1:1":{"id":1,"current":0.428,"voltage":227.0,"act_power":-38.5,"aprt_power":97.4,"pf":0.38,"freq":50.0,"calibration":"factory"},"em1data:0":{"id":0,"total_act_energy":1264.15,"total_act_ret_energy":144792.28},"em1data:1":{"id":1,"total_act_energy":48002.83,"total_act_ret_energy":33241.59},"eth":{"ip":null},"modbus":{},"mqtt":{"connected":false},"switch:0":{"id":0, "source":"HTTP_in", "output":false,"temperature":{"tC":46.4, "tF":115.5}},"sys":{"mac":"08F9E0E8AF2C","restart_required":false,"time":"10:42","unixtime":1742809323,"uptime":3671372,"ram_size":249680,"ram_free":107492,"fs_size":524288,"fs_free":188416,"cfg_rev":12,"kvs_rev":0,"schedule_rev":1,"webhook_rev":0,"available_updates":{"beta":{"version":"1.5.1-beta2"}},"reset_reason":3},"wifi":{"sta_ip":"192.168.1.120","status":"got ip","ssid":"Spaetzlewerk","rssi":-61},"ws":{"connected":false}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		// Channel 0 (1)
		assert.Equal(t, -332.2, res.Em0.ActPower)
		assert.Equal(t, 1.473, res.Em0.Current)
		assert.Equal(t, 226.9, res.Em0.Voltage)
		assert.Equal(t, 1264.15, res.Em0Data.TotalActEnergy)
		assert.Equal(t, 144792.28, res.Em0Data.TotalActRetEnergy)
		// Channel 1 (2)
		assert.Equal(t, -38.5, res.Em1.ActPower)
		assert.Equal(t, 0.428, res.Em1.Current)
		assert.Equal(t, 227.0, res.Em1.Voltage)
		assert.Equal(t, 48002.83, res.Em1Data.TotalActEnergy)
		assert.Equal(t, 33241.59, res.Em1Data.TotalActRetEnergy)
	}
}
