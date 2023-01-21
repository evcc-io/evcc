package tasmota

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// StatusResponse is the Status part of the Tasmota Status 0 command response
// https://tasmota.github.io/docs/JSON-Status-Responses/
type StatusResponse struct {
	Status struct {
		Module       int
		DeviceName   string
		FriendlyName []string
		Topic        string
		ButtonTopic  string
		Power        int
		PowerOnState int
		LedState     int
		LedMask      string
		SaveData     int
		SaveState    int
		SwitchTopic  string
		SwitchMode   []int
		ButtonRetain int
		SwitchRetain int
		SensorRetain int
		PowerRetain  int
		InfoRetain   int
		StateRetain  int
	}
}

// StatusSTSResponse is the StatusSTS part of the Tasmota Status 0 command response
// https://tasmota.github.io/docs/JSON-Status-Responses/
type StatusSTSResponse struct {
	StatusSTS struct {
		Power  string // ON, OFF, Error
		Power1 string // ON, OFF, Error
		Power2 string // ON, OFF, Error
		Power3 string // ON, OFF, Error
		Power4 string // ON, OFF, Error
		Power5 string // ON, OFF, Error
		Power6 string // ON, OFF, Error
		Power7 string // ON, OFF, Error
		Power8 string // ON, OFF, Error
	}
}

// PowerResponse is the Tasmota Power command Status response
// https://tasmota.github.io/docs/Commands/#with-web-requests
type PowerResponse struct {
	Power  string // ON, OFF, Error
	Power1 string // ON, OFF, Error
	Power2 string // ON, OFF, Error
	Power3 string // ON, OFF, Error
	Power4 string // ON, OFF, Error
	Power5 string // ON, OFF, Error
	Power6 string // ON, OFF, Error
	Power7 string // ON, OFF, Error
	Power8 string // ON, OFF, Error
}

// StatusSNSResponse is the Tasmota Status 8 command Status response
// https://tasmota.github.io/docs/JSON-Status-Responses/
type StatusSNSResponse struct {
	StatusSNS struct {
		Time string

		// Energy readings
		Energy struct {
			TotalStartTime string
			Total          float64
			Yesterday      float64
			Today          float64
			Power          Channels
			ApparentPower  Channels
			ReactivePower  Channels
			Factor         Channels
			Frequency      int
			Voltage        int
			Current        Channels
		}

		// SML sensor readings
		SML struct {
			TotalIn   float64 `json:"total_in"`
			TotalOut  float64 `json:"total_out"`
			PowerCurr int     `json:"power_curr"`
		}
	}
}

// Channels is a Tasmota specifc helper type to handle meter value lists and single meter values
type Channels []float64

func (ch *Channels) Channel(channel int) (float64, error) {
	if channel < 1 || channel > len(*ch) {
		return 0, fmt.Errorf("invalid channel: %d", channel)
	}
	return (*ch)[channel-1], nil
}

func (ch *Channels) UnmarshalJSON(data []byte) error {
	if f, err := strconv.ParseFloat(string(data), 64); err == nil {
		*ch = Channels([]float64{f})
		return nil
	}

	var ff []float64
	err := json.Unmarshal(data, &ff)
	*ch = Channels(ff)

	return err
}
