package nissan

import (
	"cmp"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Token is the Kamereon token api response
type Token struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Err returns the api error if the response carries one
func (t *Token) Err() error {
	switch {
	case t.Error == "":
		return nil
	case t.ErrorDescription == "":
		return errors.New(t.Error)
	default:
		return fmt.Errorf("%s: %s", t.Error, t.ErrorDescription)
	}
}

// Token converts the api response into an oauth2 token
func (t *Token) Token() (*oauth2.Token, error) {
	if err := t.Err(); err != nil {
		return nil, err
	}

	if t.AccessToken == "" {
		return nil, errors.New("missing access token")
	}

	expires := t.ExpiresIn
	if expires <= 0 {
		expires = 3600
	}

	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    cmp.Or(t.TokenType, "Bearer"),
		Expiry:       time.Now().Add(time.Duration(expires) * time.Second),
	}, nil
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
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
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
	ChargeStatus          float32    `json:"chargeStatus"`
	RangeHvacOff          *int       `json:"rangeHvacOff"`
	BatteryLevel          int        `json:"batteryLevel"`
	BatteryCapacity       int        `json:"batteryCapacity"`
	BatteryTemperature    int        `json:"batteryTemperature"`
	PlugStatus            int        `json:"plugStatus"`
	LastUpdateTime        *Timestamp `json:"lastUpdateTime"`
	ChargePower           int        `json:"chargePower"`
	RemainingTime         *int       `json:"chargingRemainingTime"`
	RemainingToFullFast   int        `json:"timeRequiredToFullFast"`
	RemainingToFullNormal int        `json:"timeRequiredToFullNormal"`
	RemainingToFullSlow   int        `json:"timeRequiredToFullSlow"`
	// v2
	Timestamp       *time.Time `json:"timestamp"`
	BatteryAutonomy *int       `json:"batteryAutonomy"`
	// synthesized fields
	Updated time.Time
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
		ct.Time = t
	}

	return err
}
