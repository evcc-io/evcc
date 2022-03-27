package tapo

// Tapo homepage + api reverse engineering results
// https://www.tapo.com/de/
// Credits to & inspired by:
// https://k4czp3r.xyz/reverse-engineering/tp-link/tapo/2020/10/15/reverse-engineering-tp-link-tapo.html
// https://github.com/fishbigger/TapoP100
// https://github.com/artemvang/p100-go

// Tapo connection cipher
type ConnectionCipher struct {
	Key []byte
	Iv  []byte
}

// Tapo device response
type DeviceResponse struct {
	Result struct {
		DeviceID           string `json:"device_id"`
		FWVersion          string `json:"fw_ver"`
		HWVersion          string `json:"hw_ver"`
		Type               string `json:"type"`
		Model              string `json:"model"`
		MAC                string `json:"mac"`
		HWID               string `json:"hw_id"`
		FWID               string `json:"fw_id"`
		OEMID              string `json:"oem_id"`
		Specs              string `json:"specs"`
		DeviceON           bool   `json:"device_on"`
		OnTime             int64  `json:"on_time"`
		OverHeated         bool   `json:"overheated"`
		Nickname           string `json:"nickname"`
		Location           string `json:"location"`
		Avatar             string `json:"avatar"`
		Longitude          int64  `json:"longitude"`
		Latitude           int64  `json:"latitude"`
		HasSetLocationInfo bool   `json:"has_set_location_info"`
		IP                 string `json:"ip"`
		SSID               string `json:"ssid"`
		SignalLevel        int64  `json:"signal_level"`
		RSSI               int64  `json:"rssi"`
		Region             string `json:"Europe/Kiev"`
		TimeDiff           int64  `json:"time_diff"`
		Lang               string `json:"lang"`
		Key                string `json:"key"`
		Response           string `json:"response"`
		Token              string `json:"token"`
		Current_Power      int64  `json:"current_power"`
		Today_Energy       int64  `json:"today_energy"`
	} `json:"result"`
	ErrorCode int `json:"error_code"`
}
