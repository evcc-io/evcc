package shelly

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Gen1Status response
func TestUnmarshalGen1Status(t *testing.T) {
	{
		// Shelly 1 PM channel 0 (1)
		var res Gen1Status

		jsonstr := `{"wifi_sta":{"connected":true,"ssid":"XXXX-WLAN","ip":"192.168.178.XXX","rssi":-54},"cloud":{"enabled":false,"connected":false},"mqtt":{"connected":false},"time":"17:59","unixtime":1676134770,"serial":2437,"has_update":true,"mac":"84CCA8XXXXXXX","cfg_changed_cnt":1,"actions_stats":{"skipped":0},"relays":[{"ison":false,"has_timer":false,"timer_started":0,"timer_duration":0,"timer_remaining":0,"overpower":false,"source":"http"}],"meters":[{"power":4711.12,"overpower":0.00,"is_valid":true,"timestamp":1676138370,"counters":[0.000, 0.000, 0.000],"total":6472513}],"inputs":[{"input":0,"event":"","event_cnt":0}],"temperature":16.79,"overtemperature":false,"tmp":{"tC":16.79,"tF":62.22, "is_valid":true},"temperature_status":"Normal","ext_sensors":{},"ext_temperature":{},"ext_humidity":{},"update":{"status":"pending","has_update":true,"new_version":"20221108-153925/v1.12.1-1PM-fix-g2821131","old_version":"20220209-094317/v1.11.8-g8c7bb8d"},"ram_total":50456,"ram_free":37056,"fs_size":233681,"fs_free":149094,"uptime":17284290}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		g := &gen1{model: "SHSW-PM"}
		assert.Equal(t, 107875.21666666666, g.energy(res.Meters[0].Total))
		assert.Equal(t, 4711.12, res.Meters[0].Power)
	}

	{
		// Shelly 1 channel 0 (1)
		var res Gen1Status

		jsonstr := `{"wifi_sta":{"connected":true,"ssid":"XXXX-WLAN","ip":"192.168.178.XXX","rssi":-57},"cloud":{"enabled":false,"connected":false},"mqtt":{"connected":false},"time":"19:25","unixtime":1676139913,"serial":959,"has_update":true,"mac":"E8DB8XXXXXX","cfg_changed_cnt":1,"actions_stats":{"skipped":0},"relays":[{"ison":false,"has_timer":false,"timer_started":0,"timer_duration":0,"timer_remaining":0,"source":"timer"}],"meters":[{"power":81.5,"is_valid":true}],"inputs":[{"input":0,"event":"","event_cnt":0}],"ext_sensors":{},"ext_temperature":{},"ext_humidity":{},"update":{"status":"pending","has_update":true,"new_version":"20221027-091427/v1.12.1-ga9117d3","old_version":"20211109-124958/v1.11.7-g682a0db"},"ram_total":50880,"ram_free":38796,"fs_size":233681,"fs_free":151102,"uptime":20319391}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		g := &gen1{model: "SHSW-1"}
		assert.Equal(t, 0.0, g.energy(res.Meters[0].Total))
		assert.Equal(t, 81.5, res.Meters[0].Power)
	}

	{
		// Shelly EM channel 0 (1)
		var res Gen1Status

		jsonstr := `{"wifi_sta":{"connected":true,"ssid":"XXXX","ip":"192.168.178.XXX","rssi":-55},"cloud":{"enabled":false,"connected":false},"mqtt":{"connected":false},"time":"11:16","unixtime":1676110566,"serial":21580,"has_update":false,"mac":"C45BXXXXX","cfg_changed_cnt":0,"actions_stats":{"skipped":0},"relays":[{"ison":false,"has_timer":false,"timer_started":0,"timer_duration":0,"timer_remaining":0,"overpower":false,"is_valid":true,"source":"input"}],"emeters":[{"power":-620.34,"reactive":714.48,"pf":-0.66,"voltage":235.68,"is_valid":true,"total":401472.9,"total_returned":653673.7},{"power":0.00,"reactive":0.00,"pf":0.00,"voltage":235.68,"is_valid":true,"total":173411.3,"total_returned":294.2}],"update":{"status":"idle","has_update":false,"new_version":"20221027-105518/v1.12.1-ga9117d3","old_version":"20221027-105518/v1.12.1-ga9117d3"},"ram_total":51072,"ram_free":35660,"fs_size":233681,"fs_free":156373,"uptime":2226140}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))

		g := &gen1{model: "SHEM"}
		assert.Equal(t, 401472.9, g.energy(res.EMeters[0].Total))
		assert.Equal(t, -620.34, res.EMeters[0].Power)
	}
}
