package toyota

import (
	"fmt"
	"strings"
)

type Status struct {
	Status struct {
		Messages []struct {
			ResponseCode        string `json:"responseCode"`
			Description         string `json:"description"`
			DetailedDescription string `json:"detailedDescription"`
		} `json:"messages"`
	} `json:"status"`
	Payload struct {
		BatteryLevel        int     `json:"batteryLevel"`
		ChargingStatus      string  `json:"chargingStatus"`
		EvRange             EvRange `json:"evRange"`
		EvRangeWithAc       EvRange `json:"evRangeWithAc"`
		LastUpdateTimestamp string  `json:"lastUpdateTimestamp"`
	} `json:"payload"`
}

type EvRange struct {
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

type Auth struct {
	AuthID    string         `json:"authId"`
	Callbacks []AuthCallback `json:"callbacks"`
}

type AuthCallback struct {
	Id     int8                `json:"_id"`
	Type   string              `json:"type"`
	Output []AuthCallbackValue `json:"output"`
	Input  []AuthCallbackValue `json:"input"`
}

type AuthCallbackValue struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type Token struct {
	TokenID    string `json:"tokenId"`
	SuccessURL string `json:"successUrl"`
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
	Status struct {
		Messages []struct {
			ResponseCode        string `json:"responseCode"`
			Description         string `json:"description"`
			DetailedDescription string `json:"detailedDescription"`
		} `json:"messages"`
	} `json:"status"`
	Payload []Vehicle `json:"payload"`
}

type Vehicle struct {
	VIN string `json:"vin"`
}
