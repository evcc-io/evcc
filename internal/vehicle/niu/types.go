package niu

const (
	AuthURI = "https://account-fk.niu.com"
	ApiURI  = "https://app-api-fk.niu.com"
)

// Token is the Niu oauth2 api response
// https://account-fk.niu.com/v3/api/oauth2/token?account=<NiuUser>&app_id=niu_8xt1afu6&grant_type=password&password=<MD5PasswordHash>&scope=base
type Token struct {
	Data struct {
		Token struct {
			AccessToken           string `json:"access_token,omitempty"`
			RefreshToken          string `json:"refresh_token,omitempty"`
			RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in,omitempty"`
			TokenExpiresIn        int64  `json:"token_expires_in,omitempty"`
		}
	}
}

// SoC is the Niu motor_data api response
// https://app-api-fk.niu.com/v3/motor_data/index_info?sn=<ScooterSerialNumber>
type SoC struct {
	Data struct {
		Batteries struct {
			CompartmentA struct {
				BatteryCharging int64 `json:"batteryCharging,omitempty"`
			} `json:"compartmentA"`
		}
	}
}
