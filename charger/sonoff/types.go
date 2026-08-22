package sonoff

import (
	"encoding/json"
	"fmt"
)

// Sonoff device rpc api
// https://help.sonoff.tech/docs/API_Welcome

// Request is a json-rpc request frame
type Request struct {
	Id     int    `json:"id"`
	Src    string `json:"src"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// Response is a json-rpc response frame. Error frames are returned with http status 200.
type Response struct {
	Result json.RawMessage `json:"result"`
	Error  *Error          `json:"error"`
}

// Error is a json-rpc error frame
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}

// ChannelParams addresses a single device channel
type ChannelParams struct {
	Id int `json:"id"`
}

// SetParams are the Switch.Set parameters
type SetParams struct {
	Id int  `json:"id"`
	On bool `json:"on"`
}

// SwitchStatus is the Switch.GetStatus result
// https://help.sonoff.tech/docs/API_Switch
type SwitchStatus struct {
	On bool `json:"on"`
}

// MeterStatus is the Meter.GetStatus result, scaled by 100 (energy in 0.01kWh)
// https://help.sonoff.tech/docs/API_Meter
type MeterStatus struct {
	Power       float64 `json:"power"`
	TotalEnergy float64 `json:"total_energy"`
}
