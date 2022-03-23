package tapo

import "net/http"

// Tapo homepage + api reverse engineering results
// https://www.tapo.com/de/

// https://k4czp3r.xyz/reverse-engineering/tp-link/tapo/2020/10/15/reverse-engineering-tp-link-tapo.html

type Connection struct {
	uri             string
	encodedEmail    string
	encodedPassword string
	cipher          *DeviceCipher
	sessionID       string
	token           *string
	client          *http.Client
}

type DeviceCipher struct {
	key []byte
	iv  []byte
}

type DeviceResponse struct {
	ErrorCode int `json:"error_code"`
	Result    struct {
		Key      string `json:"key"`
		Response string `json:"response"`
		Token    string `json:"token"`
	} `json:"result"`
}

type DeviceInfo struct {
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
		OnTime             int    `json:"on_time"`
		OverHeated         bool   `json:"overheated"`
		Nickname           string `json:"nickname"`
		Location           string `json:"location"`
		Avatar             string `json:"avatar"`
		Longitude          int    `json:"longitude"`
		Latitude           int    `json:"latitude"`
		HasSetLocationInfo bool   `json:"has_set_location_info"`
		IP                 string `json:"ip"`
		SSID               string `json:"ssid"`
		SignalLevel        int    `json:"signal_level"`
		RSSI               int    `json:"rssi"`
		Region             string `json:"Europe/Kiev"`
		TimeDiff           int    `json:"time_diff"`
		Lang               string `json:"lang"`
	} `json:"result"`
	ErrorCode int `json:"error_code"`
}
