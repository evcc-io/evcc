package cmd

import (
	"time"

	"github.com/andig/evcc/cmd/detect"
)

var (
	taskList = &detect.TaskList{}

	sunspecIDs   = []int{1, 2, 3, 71, 126} // modbus ids
	chargeStatus = []int{65, 66, 67}       // status values A..C
)

func init() {
	taskList.Add(detect.Task{
		ID:   "sma",
		Type: "sma",
	})

	taskList.Add(detect.Task{
		ID:   "ping",
		Type: "ping",
	})

	taskList.Add(detect.Task{
		ID:      "tcp_502",
		Type:    "tcp",
		Depends: "ping",
		Config: map[string]interface{}{
			"port": 502,
		},
	})

	taskList.Add(detect.Task{
		ID:      "sunspec",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"ids":     sunspecIDs,
			"timeout": time.Second,
		},
	})

	taskList.Add(detect.Task{
		ID:      "modbus_inverter",
		Type:    "modbus",
		Depends: "sunspec",
		Config: map[string]interface{}{
			// "port": 1502,
			"ids":     sunspecIDs,
			"models":  []int{101, 103},
			"point":   "W", // status
			"invalid": []int{65535},
			"timeout": time.Second,
		},
	})

	taskList.Add(detect.Task{
		ID:      "modbus_battery",
		Type:    "modbus",
		Depends: "sunspec",
		Config: map[string]interface{}{
			// "port": 1502,
			"ids":     sunspecIDs,
			"models":  []int{124},
			"point":   "ChaSt", // status
			"invalid": []int{65535},
			"timeout": time.Second,
		},
	})

	taskList.Add(detect.Task{
		ID:      "modbus_meter",
		Type:    "modbus",
		Depends: "sunspec",
		Config: map[string]interface{}{
			"ids":     sunspecIDs,
			"models":  []int{201, 203},
			"point":   "W",
			"timeout": time.Second,
		},
	})

	taskList.Add(detect.Task{
		ID:      "modbus_e3dc_simple",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"ids":     []int{1, 2, 3, 4, 5, 6},
			"address": 40000,
			"type":    "holding",
			"decode":  "uint16",
			"values":  []int{58332}, // 0xE3DC
		},
	})

	taskList.Add(detect.Task{
		ID:      "modbus_wallbe",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"ids":     []int{255},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	taskList.Add(detect.Task{
		ID:      "modbus_emcp",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"ids":     []int{180},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	// taskList.Add(detect.Task{
	// 	ID:      "mqtt",
	// 	Type:    "mqtt",
	// 	Depends: "ping",
	// })

	taskList.Add(detect.Task{
		ID:      "openwb",
		Type:    "mqtt",
		Depends: "ping",
		Config: map[string]interface{}{
			"topic": "openWB",
		},
	})

	taskList.Add(detect.Task{
		ID:      "tcp_80",
		Type:    "tcp",
		Depends: "ping",
		Config: map[string]interface{}{
			"port": 80,
		},
	})

	taskList.Add(detect.Task{
		ID:      "go-e",
		Type:    "http",
		Depends: "tcp_80",
		Config: map[string]interface{}{
			"path": "/status",
			"jq":   ".car",
		},
	})

	taskList.Add(detect.Task{
		ID:      "evsewifi",
		Type:    "http",
		Depends: "tcp_80",
		Config: map[string]interface{}{
			"path": "/getParameters",
			"jq":   ".type",
		},
	})

	taskList.Add(detect.Task{
		ID:      "sonnen",
		Type:    "http",
		Depends: "ping",
		Config: map[string]interface{}{
			"port": 8080,
			"path": "/api/v1/status",
			"jq":   ".GridFeedIn_W",
		},
	})

	taskList.Add(detect.Task{
		ID:      "powerwall",
		Type:    "http",
		Depends: "tcp_80",
		Config: map[string]interface{}{
			"path": "/api/meters/aggregates",
			"jq":   ".load",
		},
	})

	// // see https://github.com/andig/evcc-config/pull/5/files
	// taskList.Add(detect.Task{
	// 	ID:      "fronius",
	// 	Type:    "http",
	// 	Depends: "tcp_80",
	// 	Config: map[string]interface{}{
	// 		"path": "/solar_api/v1/GetPowerFlowRealtimeData.fcgi",
	// 		"jq":   ".Body.Data.Site.P_Grid",
	// 	},
	// })

	// taskList.Add(detect.Task{
	// 	ID:      "volkszähler",
	// 	Type:    "http",
	// 	Depends: "tcp_80",
	// 	Config: map[string]interface{}{
	// 		"path":    "/middleware.php/entity.json",
	// 		"timeout": 500 * time.Millisecond,
	// 	},
	// })
}
