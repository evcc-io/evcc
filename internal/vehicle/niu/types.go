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

// Token is the Niu oauth2 api response
// https://account-fk.niu.com/v3/api/oauth2/token?account=<NiuUser>&app_id=niu_8xt1afu6&grant_type=password&password=<MD5PasswordHash>&scope=base
type Token struct {
	Data struct {
		Token struct {
			oauth2.Token
			RefreshTokenExpiresIn int64 `json:"refresh_token_expires_in,omitempty"`
			TokenExpiresIn        int64 `json:"token_expires_in,omitempty"`
		}
	}
}

func (t *Token) UnmarshalJSON(data []byte) error {
	var s struct {
		Data struct {
			Token struct {
				oauth2.Token
				RefreshTokenExpiresIn int64 `json:"refresh_token_expires_in,omitempty"`
				TokenExpiresIn        int64 `json:"token_expires_in,omitempty"`
			}
		}
	}

	err := json.Unmarshal(data, &s)
	if err == nil {
		t.Data.Token = s.Data.Token

		if s.Data.Token.Expiry.IsZero() && s.Data.Token.TokenExpiresIn != 0 {
			t.Data.Token.Expiry = time.Now().Add(time.Second * time.Duration(s.Data.Token.TokenExpiresIn))
		}
	}

	return err
}

// Response is the Niu motor_data api response
// https://app-api-fk.niu.com/v3/motor_data/index_info?sn=<ScooterSerialNumber>
type Response struct {
	Data struct {
		IsCharging  int64 `json:"isCharging,omitempty"`
		IsConnected bool  `json:"isConnected,omitempty"`
		Timestamp   int64 `json:"time,omitempty"`
		Batteries   struct {
			CompartmentA struct {
				BatteryCharging int64 `json:"batteryCharging,omitempty"`
			} `json:"compartmentA"`
		}
		LeftTime         string `json:"leftTime,omitempty"`
		EstimatedMileage int64  `json:"estimatedMileage,omitempty"`
	}
}
