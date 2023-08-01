package niu

import (
	"encoding/json"
	"time"

	"golang.org/x/oauth2"
)

const (
	AuthURI = "https://account-fk.niu.com"
	ApiURI  = "https://app-api-fk.niu.com"
)

// https://account-fk.niu.com/v3/api/oauth2/token?account=<NiuUser>&app_id=niu_8xt1afu6&grant_type=password&password=<MD5PasswordHash>&scope=base
type Token oauth2.Token

// UnmarshalJSON decodes the token api response
func (t *Token) UnmarshalJSON(data []byte) error {
	var res struct {
		Data struct {
			Token struct {
				oauth2.Token
				RefreshTokenExpiresIn int64 `json:"refresh_token_expires_in,omitempty"`
				TokenExpiresIn        int64 `json:"token_expires_in,omitempty"`
			}
		}
	}

	err := json.Unmarshal(data, &res)
	if err == nil {
		(*t) = (Token)(res.Data.Token.Token)

		if res.Data.Token.Expiry.IsZero() && res.Data.Token.TokenExpiresIn != 0 {
			t.Expiry = time.Unix(res.Data.Token.TokenExpiresIn, 0)
		}
	}

	return err
}

// Response is the Niu motor_data api response
// https://app-api-fk.niu.com/v3/motor_data/index_info?sn=<ScooterSerialNumber>
type Response struct {
	Data struct {
		IsCharging  int   `json:"isCharging,omitempty"`
		IsConnected bool  `json:"isConnected,omitempty"`
		Timestamp   int64 `json:"time,omitempty"`
		Batteries   struct {
			CompartmentA struct {
				BatteryCharging int64 `json:"batteryCharging,omitempty"`
			} `json:"compartmentA"`
		}
		// LeftTime         float32 `json:"leftTime,omitempty"`
		EstimatedMileage int64 `json:"estimatedMileage,omitempty"`
	}
}
