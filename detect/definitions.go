package detect

import "time"

var (
	taskList = &TaskList{}

	sunspecIDs   = []int{1, 2, 3, 71, 126} // modbus ids
	chargeStatus = []int{0x41, 0x42, 0x43} // status values A..C
)

const timeout = 200 * time.Millisecond

// public task ids
const (
	TaskPing      = "ping"
	TaskTcpHttp   = "tcp_http"
	TaskTcpModbus = "tcp_modbus"
	TaskSunspec   = "sunspec"
)

// private task ids
const (
	taskOpenwb       = "openwb"
	taskSMA          = "sma"
	taskKEBA         = "KEBA"
	taskE3DC         = "e3dc_simple"
	taskSonnen       = "sonnen"
	taskPowerwall    = "powerwall"
	taskWallbe       = "wallbe"
	taskPhoenixEMEth = "phx-em-eth"
	taskPhoenixEVEth = "phx-ev-eth"
	taskEVSEWifi     = "evsewifi"
	taskGoE          = "go-e"
	taskInverter     = "inverter"
	taskBattery      = "battery"
	taskMeter        = "meter"
	taskFronius      = "fronius"
	taskTasmota      = "tasmota"
	taskTPLink       = "tplink"
)

func init() {
	// taskList.Add(Task{
	// 	ID:   taskSMA,
	// 	Type: "sma",
	// })

	// taskList.Add(Task{
	// 	ID:   taskKEBA,
	// 	Type: "keba",
	// })

	taskList.Add(Task{
		ID:   TaskPing,
		Type: "ping",
	})

	taskList.Add(Task{
		ID:      TaskTcpModbus,
		Type:    "tcp",
		Depends: TaskPing,
		Config: map[string]interface{}{
			"ports": []int{502, 1502},
		},
	})

	taskList.Add(Task{
		ID:      TaskSunspec,
		Type:    "modbus",
		Depends: TaskTcpModbus,
		Config: map[string]interface{}{
			"ids":    sunspecIDs,
			"models": []int{1},
			"point":  "Mn",
		},
	})

	// taskList.Add(Task{
	// 	ID:      taskInverter,
	// 	Type:    "modbus",
	// 	Depends: TaskSunspec,
	// 	Config: map[string]interface{}{
	// 		"ids":     sunspecIDs,
	// 		"models":  []int{101, 103},
	// 		"point":   "W",
	// 		"invalid": []int{0xFFFF},
	// 	},
	// })

	// taskList.Add(Task{
	// 	ID:      taskBattery,
	// 	Type:    "modbus",
	// 	Depends: TaskSunspec,
	// 	Config: map[string]interface{}{
	// 		"ids":     sunspecIDs,
	// 		"models":  []int{124},
	// 		"point":   "ChaSt",
	// 		"invalid": []int{0xFFFF},
	// 	},
	// })

	// taskList.Add(Task{
	// 	ID:      taskMeter,
	// 	Type:    "modbus",
	// 	Depends: TaskSunspec,
	// 	Config: map[string]interface{}{
	// 		"ids":    sunspecIDs,
	// 		"models": []int{201, 203},
	// 		"point":  "W",
	// 	},
	// })

	taskList.Add(Task{
		ID:      taskE3DC,
		Type:    "modbus",
		Depends: TaskTcpModbus,
		Config: map[string]interface{}{
			"ids":     []int{1, 2, 3, 4, 5, 6},
			"address": 40000,
			"type":    "holding",
			"decode":  "uint16",
			"values":  []int{0xE3DC},
		},
	})

	taskList.Add(Task{
		ID:      taskWallbe,
		Type:    "modbus",
		Depends: TaskTcpModbus,
		Config: map[string]interface{}{
			"ids":     []int{255},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	taskList.Add(Task{
		ID:      taskPhoenixEMEth,
		Type:    "modbus",
		Depends: TaskTcpModbus,
		Config: map[string]interface{}{
			"ids":     []int{180},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	taskList.Add(Task{
		ID:      taskPhoenixEVEth,
		Type:    "modbus",
		Depends: TaskTcpModbus,
		Config: map[string]interface{}{
			"ids":     []int{255},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	// taskList.Add(Task{
	// 	ID:      taskOpenwb,
	// 	Type:    "mqtt",
	// 	Depends: TaskPing,
	// 	Config: map[string]interface{}{
	// 		"topic": "openWB",
	// 	},
	// })

	taskList.Add(Task{
		ID:      TaskTcpHttp,
		Type:    "tcp",
		Depends: TaskPing,
		Config: map[string]interface{}{
			"ports": []int{80, 443},
		},
	})

	// taskList.Add(Task{
	// 	ID:      taskGoE,
	// 	Type:    "http",
	// 	Depends: TaskTcpHttp,
	// 	Config: map[string]interface{}{
	// 		"path": "/status",
	// 		"jq":   ".car",
	// 	},
	// })

	// taskList.Add(Task{
	// 	ID:      taskEVSEWifi,
	// 	Type:    "http",
	// 	Depends: TaskTcpHttp,
	// 	Config: map[string]interface{}{
	// 		"path": "/getParameters",
	// 		"jq":   ".type",
	// 	},
	// })

	// taskList.Add(Task{
	// 	ID:      taskSonnen,
	// 	Type:    "http",
	// 	Depends: TaskPing,
	// 	Config: map[string]interface{}{
	// 		"port": 8080,
	// 		"path": "/api/v1/status",
	// 		"jq":   ".GridFeedIn_W",
	// 	},
	// })

	// taskList.Add(Task{
	// 	ID:      taskPowerwall,
	// 	Type:    "http",
	// 	Depends: TaskTcpHttp,
	// 	Config: map[string]interface{}{
	// 		"path": "/api/meters/aggregates",
	// 		"jq":   ".load",
	// 	},
	// })

	// taskList.Add(Task{
	// 	ID:      taskFronius,
	// 	Type:    "http",
	// 	Depends: TaskTcpHttp,
	// 	Config: map[string]interface{}{
	// 		"path": "/solar_api/GetAPIVersion.cgi",
	// 		"jq":   ".BaseURL",
	// 	},
	// })

	taskList.Add(Task{
		ID:      taskTPLink,
		Type:    "tcp",
		Depends: TaskPing,
		Config: map[string]interface{}{
			"port": 9999, // TP-Link Smart Home Protocol standard port
		},
	})

	// taskList.Add(Task{
	// 	ID:      taskTasmota,
	// 	Type:    "http",
	// 	Depends: TaskTcpHttp,
	// 	Config: map[string]interface{}{
	// 		"path": "//cm?cmnd=Module",
	// 		"jq":   ".Module",
	// 	},
	// })

	taskList.Add(Task{
		ID:      "volkszähler",
		Type:    "http",
		Depends: TaskTcpHttp,
		Config: map[string]interface{}{
			"path":    "/middleware.php/entity.json",
			"timeout": 500 * time.Millisecond,
		},
	})
}
