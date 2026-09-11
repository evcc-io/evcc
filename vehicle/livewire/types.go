package livewire

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Response shapes follow API.md (app v1.8.0). Fields marked provisional have not
// been confirmed against live samples yet.

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

// SessionResponse carries the LiveWire JWT. The field name is provisional, hence
// the candidates.
type SessionResponse struct {
	Envelope
	Token       string `json:"token"`
	JWT         string `json:"jwt"`
	AccessToken string `json:"accessToken"`
}

// Jwt returns the first populated token candidate
func (r SessionResponse) Jwt() string {
	for _, s := range []string{r.Token, r.JWT, r.AccessToken} {
		if s != "" {
			return s
		}
	}
	return ""
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
// may arrive with HTTP 200
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
	ID    string `json:"id"`
	VIN   string `json:"vin"`
	Model string `json:"model"`
}

// BikesResponse accepts both a bare array and an object layout (provisional)
type BikesResponse struct {
	Envelope
	Bikes []Bike `json:"bikes"`
}

func (r *BikesResponse) UnmarshalJSON(b []byte) error {
	if strings.HasPrefix(strings.TrimSpace(string(b)), "[") {
		return json.Unmarshal(b, &r.Bikes)
	}

	type plain BikesResponse
	return json.Unmarshal(b, (*plain)(r))
}

// ChargingStatus is the bike charging data (provisional, see API.md §4.2)
type ChargingStatus struct {
	ChargingStatus    bool        `json:"chargingStatus"`
	PluggedIn         bool        `json:"pluggedIn"`
	BatteryPercentage StringFloat `json:"batteryPercentage"`
	Range             float64     `json:"range"`
	TimeToMaxLimit    int64       `json:"timeToMaxLimit"`
	MaxLimit          int64       `json:"maxLimit"`
	Odometer          float64     `json:"odometer"`
	DurationElapsed   int64       `json:"durationElapsed"`
	DurationUnit      string      `json:"durationUnit"`
}

// ChargingStatusResponse decodes both the flat and the wrapped layout, since
// API.md names the type ChargingStatusResponse.BikeChargingData
type ChargingStatusResponse struct {
	Envelope
	ChargingStatus
	BikeChargingData *ChargingStatus `json:"bikeChargingData"`
}

// Data returns the charging data regardless of layout
func (r ChargingStatusResponse) Data() ChargingStatus {
	if r.BikeChargingData != nil {
		return *r.BikeChargingData
	}
	return r.ChargingStatus
}

// Position is the bike location (provisional)
type Position struct {
	Envelope
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// StringFloat decodes a number that may arrive as JSON number or string,
// optionally with a trailing percent sign
type StringFloat float64

func (f *StringFloat) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == `""` {
		return nil
	}

	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		s = strings.TrimSuffix(strings.TrimSpace(str), "%")
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid number %q: %w", string(b), err)
	}

	*f = StringFloat(v)
	return nil
}
