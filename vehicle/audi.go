package vehicle

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/vwidentity"
	"golang.org/x/net/publicsuffix"
)

const (
	vwAPI = "https://msg.volkswagen.de/fs-car"
)

// OIDCResponse is the well-known OIDC provider response
// https://{oauth-provider-hostname}/.well-known/openid-configuration
type OIDCResponse struct {
	Issuer      string   `json:"issuer"`
	AuthURL     string   `json:"authorization_endpoint"`
	TokenURL    string   `json:"token_endpoint"`
	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}

type audiTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type audiVehiclesResponse struct {
	UserVehicles struct {
		Vehicle []string
	}
}

type audiChargerResponse struct {
	Charger struct {
		Status struct {
			BatteryStatusData struct {
				StateOfCharge struct {
					Content   int
					Timestamp string
				}
				RemainingChargingTime struct {
					Content   int
					Timestamp string
				}
			}
		}
	}
}

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*request.Helper
	user, password, vin string
	brand, country      string
	tokens              audiTokenResponse
	chargeStateG        func() (float64, error)
	finishTimeG         func() (time.Time, error)
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("audi")

	v := &Audi{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
		brand:    "Audi",
		country:  "DE",
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()
	v.finishTimeG = provider.NewCached(v.finishTime, cc.Cache).TimeGetter()

	var err error
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// track cookies and don't follow redirects
	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	if err == nil {
		err = v.authFlow()
	}

	if err == nil && cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, err
}

func (v *Audi) authFlow() error {
	var uri, body string
	var req *http.Request

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?" +
		"response_type=code&client_id=09b6cbec-cd19-4589-82fd-363dfa8c24da%40apps_vw-dilab_com&" +
		"redirect_uri=myaudi%3A%2F%2F%2F&scope=address%20profile%20badge%20birthdate%20birthplace%20nationalIdentifier%20nationality%20profession%20email%20vin%20phone%20nickname%20name%20picture%20mbb%20gallery%20openid&" +
		"state=7f8260b5-682f-4db8-b171-50a5189a1c08&nonce=583b9af2-7799-4c72-9cb0-e6c0f42b87b3&prompt=login&ui_locales=de-DE"

	identity := &vwidentity.Identity{Client: v.Client}
	resp, err := identity.Login(uri, v.user, v.password)

	var tokens audiTokenResponse
	if err == nil {
		var code string
		if location, err := url.Parse(resp.Header.Get("Location")); err == nil {
			code = location.Query().Get("code")
		}

		data := url.Values(map[string][]string{
			"client_id":     {"09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"},
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {"myaudi:///"},
			"response_type": {"token id_token"},
		})

		uri = "https://app-api.my.audi.com/myaudiappidk/v1/token"
		req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
		if err == nil {
			err = v.DoJSON(req, &tokens)
		}
	}

	if err == nil {
		body = fmt.Sprintf("grant_type=%s&token=%s&scope=%s", "id_token", tokens.IDToken, "sc2:fal")
		headers := map[string]string{
			"Content-Type":  "application/x-www-form-urlencoded",
			"X-App-Version": "3.14.0",
			"X-App-Name":    "myAudi",
			"X-Client-Id":   "77869e21-e30a-4a92-b016-48ab7d3db1d8",
		}

		req, err = request.New(http.MethodPost, vwidentity.OauthTokenURI, strings.NewReader(body), headers)
		if err == nil {
			err = v.DoJSON(req, &tokens)
			v.tokens = tokens
		}
	}

	return err
}

func (v *Audi) refreshToken() error {
	if v.tokens.RefreshToken == "" {
		return errors.New("missing refresh token")
	}

	body := fmt.Sprintf("grant_type=%s&refresh_token=%s&scope=%s", "refresh_token", v.tokens.RefreshToken, "sc2:fal")
	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"X-App-Version": "3.14.0",
		"X-App-Name":    "myAudi",
		"X-Client-Id":   "77869e21-e30a-4a92-b016-48ab7d3db1d8",
	}

	req, err := request.New(http.MethodPost, vwidentity.OauthTokenURI, strings.NewReader(body), headers)
	if err == nil {
		var tokens audiTokenResponse
		err = v.DoJSON(req, &tokens)
		if err == nil {
			v.tokens = tokens
		}
	}

	return err
}

func (v *Audi) getJSON(uri string, res interface{}) error {
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.tokens.AccessToken,
	})

	if err == nil {
		err = v.DoJSON(req, &res)

		// token expired?
		if err != nil {
			resp := v.LastResponse()

			// handle http 401
			if resp != nil && resp.StatusCode == http.StatusUnauthorized {
				// use refresh token
				err = v.refreshToken()

				// re-run auth flow
				if err != nil {
					err = v.authFlow()
				}
			}

			// retry original requests
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+v.tokens.AccessToken)
				err = v.DoJSON(req, &res)
			}
		}
	}

	return err
}

func (v *Audi) vehicles() ([]string, error) {
	var res audiVehiclesResponse
	uri := fmt.Sprintf("%s/usermanagement/users/v1/Audi/DE/vehicles", vwAPI)
	err := v.getJSON(uri, &res)
	return res.UserVehicles.Vehicle, err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Audi) chargeState() (float64, error) {
	var res audiChargerResponse
	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", vwAPI, v.brand, v.country, v.vin)
	err := v.getJSON(uri, &res)
	return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Audi) ChargeState() (float64, error) {
	return v.chargeStateG()
}

// finishTime implements the Vehicle.ChargeFinishTimer interface
func (v *Audi) finishTime() (time.Time, error) {
	var res audiChargerResponse
	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", vwAPI, v.brand, v.country, v.vin)
	err := v.getJSON(uri, &res)

	var timestamp time.Time
	if err == nil {
		timestamp, err = time.Parse(time.RFC3339, res.Charger.Status.BatteryStatusData.RemainingChargingTime.Timestamp)
	}

	return timestamp.Add(time.Duration(res.Charger.Status.BatteryStatusData.RemainingChargingTime.Content) * time.Minute), err
}

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *Audi) FinishTime() (time.Time, error) {
	return v.finishTimeG()
}
