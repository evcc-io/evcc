package nissan

import (
	"fmt"
	"strings"
	"time"
)

type Auth struct {
	AuthID    string         `json:"authId"`
	Template  string         `json:"template"`
	Stage     string         `json:"stage"`
	Header    string         `json:"header"`
	Callbacks []AuthCallback `json:"callbacks"`
}

type AuthCallback struct {
	Type   string              `json:"type"`
	Output []AuthCallbackValue `json:"output"`
	Input  []AuthCallbackValue `json:"input"`
}

type AuthCallbackValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Token struct {
	TokenID    string `json:"tokenId"`
	SuccessURL string `json:"successUrl"`
	Realm      string `json:"realm"`
	Code       int    `json:"code"`    // error response
	Reason     string `json:"reason"`  // error response
	Message    string `json:"message"` // error response
}

func (t *Token) SessionExpired() bool {
	return strings.EqualFold(t.Message, "Session has timed out")
}

func (t *Token) Error() error {
	if t.Code == 0 {
		return nil
	}
	return fmt.Errorf("%s: %s", t.Reason, t.Message)
}

type Vehicles struct {
	Data []Vehicle
}

type Vehicle struct {
	VIN        string
	ModelName  string
	PictureURL string
}

// Request structure for kamereon api
type Request struct {
	Data Payload `json:"data"`
}

type Payload struct {
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type Error struct {
	Status, Code, Detail string
}

// StatusResponse structure for kamereon api
type StatusResponse struct {
	ID string
	Attributes
	Errors []Error
}

type Attributes struct {
	ChargeStatus          float32   `json:"chargeStatus"`
	RangeHvacOff          *int      `json:"rangeHvacOff"`
	BatteryLevel          int       `json:"batteryLevel"`
	BatteryCapacity       int       `json:"batteryCapacity"`
	BatteryTemperature    int       `json:"batteryTemperature"`
	PlugStatus            int       `json:"plugStatus"`
	LastUpdateTime        Timestamp `json:"lastUpdateTime"`
	ChargePower           int       `json:"chargePower"`
	RemainingTime         *int      `json:"chargingRemainingTime"`
	RemainingToFullFast   int       `json:"timeRequiredToFullFast"`
	RemainingToFullNormal int       `json:"timeRequiredToFullNormal"`
	RemainingToFullSlow   int       `json:"timeRequiredToFullSlow"`
}

type ActionResponse struct {
	Data struct {
		Type, ID string // battery refresh
	} `json:"data"`
	Errors []Error
}

const timeFormat = "2006-01-02T15:04:05Z"

// Timestamp implements JSON unmarshal
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes string timestamp into time.Time
func (ct *Timestamp) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")

	t, err := time.Parse(timeFormat, s)
	if err == nil {
		(*ct).Time = t
	}

	return err
}
