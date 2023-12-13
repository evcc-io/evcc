package shelly

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Gen1StatusResponse response
func TestUnmarshalGen1StatusResponse(t *testing.T) {
	{
		// Shelly 1 PM channel 0 (1)
		var res Gen1StatusResponse

		jsonstr := `{"wifi_sta":{"connected":true,"ssid":"XXXX-WLAN","ip":"192.168.178.XXX","rssi":-54},"cloud":{"enabled":false,"connected":false},"mqtt":{"connected":false},"time":"17:59","unixtime":1676134770,"serial":2437,"has_update":true,"mac":"84CCA8XXXXXXX","cfg_changed_cnt":1,"actions_stats":{"skipped":0},"relays":[{"ison":false,"has_timer":false,"timer_started":0,"timer_duration":0,"timer_remaining":0,"overpower":false,"source":"http"}],"meters":[{"power":4711.12,"overpower":0.00,"is_valid":true,"timestamp":1676138370,"counters":[0.000, 0.000, 0.000],"total":6472513}],"inputs":[{"input":0,"event":"","event_cnt":0}],"temperature":16.79,"overtemperature":false,"tmp":{"tC":16.79,"tF":62.22, "is_valid":true},"temperature_status":"Normal","ext_sensors":{},"ext_temperature":{},"ext_humidity":{},"update":{"status":"pending","has_update":true,"new_version":"20221108-153925/v1.12.1-1PM-fix-g2821131","old_version":"20220209-094317/v1.11.8-g8c7bb8d"},"ram_total":50456,"ram_free":37056,"fs_size":233681,"fs_free":149094,"uptime":17284290}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, 107875.21666666666, gen1Energy("SHSW-PM", res.Meters[0].Total))
		assert.Equal(t, 4711.12, res.Meters[0].Power)
	}

	{
		// Shelly 1 channel 0 (1)
		var res Gen1StatusResponse

		jsonstr := `{"wifi_sta":{"connected":true,"ssid":"XXXX-WLAN","ip":"192.168.178.XXX","rssi":-57},"cloud":{"enabled":false,"connected":false},"mqtt":{"connected":false},"time":"19:25","unixtime":1676139913,"serial":959,"has_update":true,"mac":"E8DB8XXXXXX","cfg_changed_cnt":1,"actions_stats":{"skipped":0},"relays":[{"ison":false,"has_timer":false,"timer_started":0,"timer_duration":0,"timer_remaining":0,"source":"timer"}],"meters":[{"power":81.5,"is_valid":true}],"inputs":[{"input":0,"event":"","event_cnt":0}],"ext_sensors":{},"ext_temperature":{},"ext_humidity":{},"update":{"status":"pending","has_update":true,"new_version":"20221027-091427/v1.12.1-ga9117d3","old_version":"20211109-124958/v1.11.7-g682a0db"},"ram_total":50880,"ram_free":38796,"fs_size":233681,"fs_free":151102,"uptime":20319391}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, 0.0, gen1Energy("SHSW-1", res.Meters[0].Total))
		assert.Equal(t, 81.5, res.Meters[0].Power)
	}

	{
		// Shelly EM channel 0 (1)
		var res Gen1StatusResponse

		jsonstr := `{"wifi_sta":{"connected":true,"ssid":"XXXX","ip":"192.168.178.XXX","rssi":-55},"cloud":{"enabled":false,"connected":false},"mqtt":{"connected":false},"time":"11:16","unixtime":1676110566,"serial":21580,"has_update":false,"mac":"C45BXXXXX","cfg_changed_cnt":0,"actions_stats":{"skipped":0},"relays":[{"ison":false,"has_timer":false,"timer_started":0,"timer_duration":0,"timer_remaining":0,"overpower":false,"is_valid":true,"source":"input"}],"emeters":[{"power":-620.34,"reactive":714.48,"pf":-0.66,"voltage":235.68,"is_valid":true,"total":401472.9,"total_returned":653673.7},{"power":0.00,"reactive":0.00,"pf":0.00,"voltage":235.68,"is_valid":true,"total":173411.3,"total_returned":294.2}],"update":{"status":"idle","has_update":false,"new_version":"20221027-105518/v1.12.1-ga9117d3","old_version":"20221027-105518/v1.12.1-ga9117d3"},"ram_total":51072,"ram_free":35660,"fs_size":233681,"fs_free":156373,"uptime":2226140}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		assert.Equal(t, 401472.9, gen1Energy("SHEM", res.EMeters[0].Total))
		assert.Equal(t, -620.34, res.EMeters[0].Power)
	}
}

// Test Gen2StatusResponse response
func TestUnmarshalGen2StatusResponse(t *testing.T) {
	// Shelly Pro 1PM channel 0 (1)
	var res Gen2StatusResponse

	jsonstr := `{"ble":{},"cloud":{"connected":true},"eth":{"ip":null},"input:0":{"id":0,"state":false},"input:1":{"id":1,"state":false},"mqtt":{"connected":false},"switch:0":{"id":0, "source":"HTTP", "output":false, "apower":47.11, "voltage":232.0, "current":0.000, "pf":0.00, "aenergy":{"total":5.125,"by_minute":[0.000,0.000,0.000],"minute_ts":1675718520},"temperature":{"tC":25.3, "tF":77.5}},"sys":{"mac":"30C6F78BB4D8","restart_required":false,"time":"22:22","unixtime":1675718522,"uptime":45070,"ram_size":234204,"ram_free":137716,"fs_size":524288,"fs_free":172032,"cfg_rev":13,"kvs_rev":1,"schedule_rev":0,"webhook_rev":0,"available_updates":{"beta":{"version":"0.13.0-beta3"}}},"wifi":{"sta_ip":"192.168.178.64","status":"got ip","ssid":"***","rssi":-62},"ws":{"connected":false}}`
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

	assert.Equal(t, 5.125, res.Switch0.Aenergy.Total)
	assert.Equal(t, 47.11, res.Switch0.Apower)
}
