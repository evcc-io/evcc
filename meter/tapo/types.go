package tapo

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/evcc-io/evcc/util"
	"github.com/google/uuid"
)

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

type tapoError int

func (te tapoError) Error() string {
	switch te {
	case 0:
		return "Success"
	case -1010:
		return "Invalid Public Key Length"
	case -1012:
		return "Invalid terminalUUID"
	case -1501:
		return "Invalid Request or Credentials"
	case 1002:
		return "Incorrect Request"
	case -1003:
		return "JSON formatting error"
	case 1003:
		return "Communication error"
	case 9999:
		return "Session timeout"
	default:
		return fmt.Sprintf("Unknown error: %d", te)
	}
}

type Plug struct {
	log          *util.Logger
	Addr         netip.Addr
	terminalUUID uuid.UUID
	session      Session
}

type LoginDeviceRequest struct {
	Method          string `json:"method"`
	RequestTimeMils int    `json:"requestTimeMils"`
	Params          struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"params"`
}

type LoginDeviceResponse struct {
	ErrorCode tapoError `json:"error_code"`
	Result    struct {
		Token string `json:"token"`
	} `json:"result"`
}

type Session interface {
	Handshake(addr netip.Addr, username, password string) error
	Request([]byte) ([]byte, error)
	Addr() netip.Addr
}
type HandshakeRequest struct {
	Method          string `json:"method"`
	RequestTimeMils int    `json:"requestTimeMils"`
	Params          struct {
		Key string `json:"key"`
	} `json:"params"`
}

type HandshakeResponse struct {
	ErrorCode tapoError `json:"error_code"`
	Result    struct {
		Key string `json:"key"`
	}
}

type SecurePassthroughRequest struct {
	Method string `json:"method"`
	Params struct {
		Request string `json:"request"`
	} `json:"params"`
}

func NewSecurePassthroughRequest(innerRequest string) *SecurePassthroughRequest {
	r := SecurePassthroughRequest{
		Method: "securePassthrough",
	}
	r.Params.Request = innerRequest
	return &r
}

type SecurePassthroughResponse struct {
	ErrorCode tapoError `json:"error_code"`
	Result    struct {
		Response string `json:"response"`
	}
}

type DeviceInfo struct {
	DeviceID           string `json:"device_id"`
	FWVersion          string `json:"fw_ver"`
	HWVersion          string `json:"hw_ver"`
	Type               string `json:"type"`
	Model              string `json:"model"`
	MAC                string `json:"mac"`
	HWID               string `json:"hw_id"`
	FWID               string `json:"fw_id"`
	OEMID              string `json:"oem_id"`
	IP                 string `json:"ip"`
	TimeDiff           int    `json:"time_diff"`
	SSID               string `json:"ssid"`
	RSSI               int    `json:"rssi"`
	SignalLevel        int    `json:"signal_level"`
	Latitude           int    `json:"latitude"`
	Longitude          int    `json:"longitude"`
	Lang               string `json:"lang"`
	Avatar             string `json:"avatar"`
	Region             string `json:"region"`
	Specs              string `json:"specs"`
	Nickname           string `json:"nickname"`
	HasSetLocationInfo bool   `json:"has_set_location_info"`
	DeviceON           bool   `json:"device_on"`
	OnTime             int    `json:"on_time"`
	DefaultStates      struct {
		Type string `json:"type"`
		// TODO add the structure for State
		State *json.RawMessage `json:"state"`
	} `json:"default_states"`
	OverHeated            bool   `json:"overheated"`
	PowerProtectionStatus string `json:"power_protection_status,omitempty"`
	Location              string `json:"location,omitempty"`

	// Computed values below.
	// DecodedSSID is the decoded version of the base64-encoded SSID field.
	DecodedSSID string
	// DecodedNickname is the decoded version of the base64-encoded Nickname field.
	DecodedNickname string
}

type GetDeviceInfoResponse struct {
	ErrorCode tapoError  `json:"error_code"`
	Result    DeviceInfo `json:"result"`
}

type GetDeviceInfoRequest struct {
	Method          string `json:"method"`
	RequestTimeMils int    `json:"requestTimeMils"`
}

type SetDeviceInfoRequest struct {
	Method string `json:"method"`
	Params struct {
		DeviceOn bool `json:"device_on"`
	} `json:"params"`
}

type SetDeviceInfoResponse struct {
	ErrorCode tapoError `json:"error_code"`
	Result    struct {
		Response string `json:"response"`
	}
}

type DeviceUsage struct {
	TimeUsage struct {
		Today  int `json:"today"`
		Past7  int `json:"past7"`
		Past30 int `json:"past30"`
	} `json:"time_usage"`
	PowerUsage struct {
		Today  int `json:"today"`
		Past7  int `json:"past7"`
		Past30 int `json:"past30"`
	} `json:"power_usage"`
	SavedPower struct {
		Today  int `json:"today"`
		Past7  int `json:"past7"`
		Past30 int `json:"past30"`
	} `json:"saved_power"`
}

type GetDeviceUsageRequest struct {
	Method          string `json:"method"`
	RequestTimeMils int    `json:"requestTimeMils"`
}

type GetDeviceUsageResponse struct {
	ErrorCode tapoError   `json:"error_code"`
	Result    DeviceUsage `json:"result"`
}

type EnergyUsage struct {
	TodayRuntime      int    `json:"today_runtime"`
	MonthRuntime      int    `json:"month_runtime"`
	TodayEnergy       int64  `json:"today_energy"`
	MonthEnergy       int64  `json:"month_energy"`
	LocalTime         string `json:"local_time"`
	ElectricityCharge [3]int `json:"electricity_charge"`
	CurrentPower      int64  `json:"current_power"`
}

type GetEnergyUsageRequest struct {
	Method          string `json:"method"`
	RequestTimeMils int    `json:"requestTimeMils"`
}

type GetEnergyUsageResponse struct {
	ErrorCode tapoError   `json:"error_code"`
	Result    EnergyUsage `json:"result"`
}
