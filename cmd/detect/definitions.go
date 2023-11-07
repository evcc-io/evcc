package detect

import (
	"github.com/evcc-io/evcc/cmd/detect/tasks"
)

var (
	taskList = &TaskList{}

	sunspecIDs   = []int{1, 2, 3, 71, 126, 200, 201, 202, 203, 204, 240} // modbus ids
	chargeStatus = []int{0x41, 0x42, 0x43}                               // status values A..C
)

// public task ids
const (
	TaskPing    = "ping"
	TaskHttp    = "tcp_http"
	TaskModbus  = "tcp_modbus"
	TaskSunspec = "sunspec"
)

// private task ids
const (
	taskOpenwb       = "openwb"
	taskSMA          = "sma"
	taskKEBA         = "keba"
	taskE3DC         = "e3dc_simple"
	taskSonnen       = "sonnen"
	taskPowerwall    = "powerwall"
	taskWallbe       = "wallbe"
	taskPhoenixEMEth = "phx-em-eth"
	taskPhoenixEVEth = "phx-ev-eth"
	taskEVSEWifi     = "evsewifi"
	taskGoE          = "go-e"
	taskInverter     = "inverter"
	taskStrings      = "strings"
	taskBattery      = "battery"
	taskMeter        = "meter"
	taskFroniusWeb   = "fronius-web"
	taskTasmota      = "tasmota"
	taskShelly       = "shelly"
	// taskTPLink       = "tplink"
)

func init() {
	taskList.Add(tasks.Task{
		ID:   TaskPing,
		Type: tasks.Ping,
	})

	taskList.Add(tasks.Task{
		ID:      taskSMA,
		Type:    tasks.Sma,
		Depends: TaskPing,
	})

	taskList.Add(tasks.Task{
		ID:      taskKEBA,
		Type:    tasks.Keba,
		Depends: TaskPing,
	})

	taskList.Add(tasks.Task{
		ID:      TaskModbus,
		Type:    tasks.Tcp,
		Depends: TaskPing,
		Config: map[string]interface{}{
			"ports": []int{502, 1502},
		},
	})

	taskList.Add(tasks.Task{
		ID:      TaskSunspec,
		Type:    tasks.Modbus,
		Depends: TaskModbus,
		Config: map[string]interface{}{
			"ids":    sunspecIDs,
			"models": []int{1},
			"point":  "Mn",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskInverter,
		Type:    tasks.Modbus,
		Depends: TaskSunspec,
		Config: map[string]interface{}{
			"ids":     sunspecIDs,
			"models":  []int{101, 103},
			"point":   "W",
			"invalid": []int{0xFFFF},
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskStrings,
		Type:    tasks.Modbus,
		Depends: TaskSunspec,
		Config: map[string]interface{}{
			"ids":     sunspecIDs,
			"models":  []int{160},
			"point":   "N",
			"invalid": []int{0xFFFF},
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskBattery,
		Type:    tasks.Modbus,
		Depends: TaskSunspec,
		Config: map[string]interface{}{
			"ids":     sunspecIDs,
			"models":  []int{124},
			"point":   "ChaSt",
			"invalid": []int{0xFFFF},
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskMeter,
		Type:    tasks.Modbus,
		Depends: TaskSunspec,
		Config: map[string]interface{}{
			"ids":    sunspecIDs,
			"models": []int{201, 203, 211, 213},
			"point":  "W",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskE3DC,
		Type:    tasks.Modbus,
		Depends: TaskModbus,
		Config: map[string]interface{}{
			"ids":     []int{1, 2, 3, 4, 5, 6},
			"address": 40000,
			"type":    "holding",
			"decode":  "uint16",
			"values":  []int{0xE3DC},
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskWallbe,
		Type:    tasks.Modbus,
		Depends: TaskModbus,
		Config: map[string]interface{}{
			"ids":     []int{255},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskPhoenixEMEth,
		Type:    tasks.Modbus,
		Depends: TaskModbus,
		Config: map[string]interface{}{
			"ids":     []int{180},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskPhoenixEVEth,
		Type:    tasks.Modbus,
		Depends: TaskModbus,
		Config: map[string]interface{}{
			"ids":     []int{255},
			"address": 100,
			"type":    "input",
			"decode":  "uint16",
			"values":  chargeStatus,
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskOpenwb,
		Type:    tasks.Mqtt,
		Depends: TaskPing,
		Config: map[string]interface{}{
			"topic": "openWB",
		},
	})

	taskList.Add(tasks.Task{
		ID:      TaskHttp,
		Type:    tasks.Tcp,
		Depends: TaskPing,
		Config: map[string]interface{}{
			"ports": []int{80, 443},
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskGoE,
		Type:    tasks.Http,
		Depends: TaskHttp,
		Config: map[string]interface{}{
			"path": "/status",
			"jq":   ".car",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskEVSEWifi,
		Type:    tasks.Http,
		Depends: TaskHttp,
		Config: map[string]interface{}{
			"path": "/getParameters",
			"jq":   ".type",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskSonnen,
		Type:    tasks.Http,
		Depends: TaskPing,
		Config: map[string]interface{}{
			"port": 8080,
			"path": "/api/v1/status",
			"jq":   ".GridFeedIn_W",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskPowerwall,
		Type:    tasks.Http,
		Depends: TaskHttp,
		Config: map[string]interface{}{
			"path": "/api/meters/aggregates",
			"jq":   ".load",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskFroniusWeb,
		Type:    tasks.Http,
		Depends: TaskHttp,
		Config: map[string]interface{}{
			"path": "/solar_api/GetAPIVersion.cgi",
			"jq":   ".BaseURL",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskTasmota,
		Type:    tasks.Http,
		Depends: TaskHttp,
		Config: map[string]interface{}{
			"path": "/cm?cmnd=Module",
			"jq":   ".Module",
		},
	})

	// taskList.Add(tasks.Task{
	// 	ID:      taskTPLink,
	// 	Type:    tasks.Http,
	// 	Depends: TaskHttp,
	// 	Config: map[string]interface{}{
	// 		"ResponseHeader": map[string]string{
	// 			"Server": "TP-LINK Smart Plug",
	// 		},
	// 	},
	// })

	taskList.Add(tasks.Task{
		ID:      "volkszähler",
		Type:    tasks.Http,
		Depends: TaskHttp,
		Config: map[string]interface{}{
			"path": "/middleware.php/entity.json",
			"jq":   ".version",
		},
	})

	taskList.Add(tasks.Task{
		ID:      taskShelly,
		Type:    tasks.Http,
		Depends: TaskHttp,
		Config: map[string]interface{}{
			"path": "/shelly",
			"jq":   ".type",
		},
	})
}
