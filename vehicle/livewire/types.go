package livewire

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Response shapes confirmed against the live backend on 2026-09-11 (app v1.8.0).

type gigyaResponse struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	UID          string `json:"UID"`
}

type SessionRequest struct {
	UID        string `json:"UID"`
	DeviceUUID string `json:"deviceUUID"`
	DataCenter string `json:"dataCenter"`
}

type SessionResponse struct {
	Envelope
	JWT           string `json:"jwt"`
	TermsAccepted bool   `json:"termsAccepted"`
}

// Error is the api error envelope content
type Error struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s (%s)", e.Description, e.Code)
}

// Envelope is embedded in every response to detect the error envelope, which
// arrives with HTTP 200
type Envelope struct {
	Error *Error `json:"error"`
}

// Err returns the envelope error, if any
func (e Envelope) Err() error {
	if e.Error == nil {
		return nil
	}
	return e.Error
}

type Bike struct {
	ID            string `json:"id"`
	VIN           string `json:"vin"`
	Make          string `json:"make"`
	Model         string `json:"model"`
	Name          string `json:"name"`
	PairingStatus bool   `json:"pairingStatus"` // only present in the pairingStatus response
}

type BikesResponse struct {
	Envelope
	Bikes []Bike `json:"bikes"`
}

type ChargingStatus struct {
	ChargingStatus    bool        `json:"chargingStatus"`
	PluggedIn         bool        `json:"pluggedIn"`
	BatteryPercentage float64     `json:"batteryPercentage"`
	Range             float64     `json:"range"`          // miles
	TimeToMaxLimit    StringFloat `json:"timeToMaxLimit"` // "0.0"
	MaxLimit          int64       `json:"maxLimit"`
	Odometer          float64     `json:"odometer"`        // miles
	DurationElapsed   StringFloat `json:"durationElapsed"` // "4"
	DurationUnit      string      `json:"durationUnit"`    // "seconds", refers to durationElapsed
}

type ChargingStatusResponse struct {
	Envelope
	BikeChargingData ChargingStatus `json:"bikeChargingData"`
}

type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
	SpeedKm   float64 `json:"speed_km"`
	GpsTime   string  `json:"gps_time"`
	FixType   int     `json:"fix_type"`
}

type LocationResponse struct {
	Envelope
	Data Position `json:"data"`
}

// StringFloat decodes a number that arrives as JSON string or number
type StringFloat float64

func (f *StringFloat) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == `""` {
		return nil
	}

	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		s = strings.TrimSpace(str)
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid number %q: %w", string(b), err)
	}

	*f = StringFloat(v)
	return nil
}
