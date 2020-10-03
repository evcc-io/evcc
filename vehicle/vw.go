package vehicle

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/vw"
	"golang.org/x/net/publicsuffix"
)

// https://github.com/trocotronic/weconnect

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	*request.Helper
	user, password string
	clientID       string
	tokens         vw.Tokens
	api            *vw.API
	apiG           func() (interface{}, error)
}

func init() {
	registry.Add("vw", NewVWFromConfig)
}

// NewVWFromConfig creates a new vehicle
func NewVWFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("vw")

	v := &VW{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
	}

	v.api = vw.NewAPI(v.Helper, &v.tokens, v.authFlow, v.refreshHeaders, strings.ToUpper(cc.VIN), "VW", "DE")
	v.apiG = provider.NewCached(v.apiCall, cc.Cache).InterfaceGetter()

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
		v.api.VIN, err = findVehicle(v.api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.api.VIN)
		}
	}

	return v, err
}

func (v *VW) authFlow() error {
	var err error
	var uri string
	var req *http.Request
	var resp *http.Response
	var idToken string

	// execute login
	challenge, verifier, err := vw.ChallengeVerifier()
	query := url.Values(map[string][]string{
		"prompt":                {"login"},
		"state":                 {vw.RandomString(43)},
		"response_type":         {"code id_token token"},
		"code_challenge_method": {"s256"},
		"scope":                 {"openid profile mbb cars birthdate nickname address phone"},
		"code_challenge":        {challenge},
		"redirect_uri":          {"carnet://identity-kit/login"},
		"client_id":             {"9496332b-ea03-4091-a224-8c746b885068@apps_vw-dilab_com"},
		"nonce":                 {vw.RandomString(43)},
	})

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?" + query.Encode()
	if err == nil {
		identity := &vw.Identity{Client: v.Client}
		resp, err = identity.Login(uri, v.user, v.password)
	}

	if err == nil {
		// var code string

		loc := strings.ReplaceAll(resp.Header.Get("Location"), "#", "?") //  convert to parsable url
		if locationURL, err := url.Parse(loc); err == nil {
			// code = locationURL.Query().Get("code")
			idToken = locationURL.Query().Get("id_token")
		}

		_ = verifier
		// if err == nil {
		// 	data := url.Values(map[string][]string{
		// 		"auth_code":     {code},
		// 		"code_verifier": {verifier},
		// 		"id_token":      {idToken},
		// 	})

		// 	uri := "https://tokenrefreshservice.apps.emea.vwapps.io/exchangeAuthCode"
		// 	req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

		// 	// if err == nil {
		// 	// 	err = v.DoJSON(req, &tokens)
		// 	// 	if err == nil && tokens.AccessToken == "" {
		// 	// 		err = errors.New("missing access token")
		// 	// 	}
		// 	// }
		// }
	}

	// get client id
	if err == nil {
		data := struct {
			AppID       string `json:"appId"`
			AppName     string `json:"appName"`
			AppVersion  string `json:"appVersion"`
			ClientBrand string `json:"client_brand"`
			ClientName  string `json:"client_name"`
			Platform    string `json:"platform"`
		}{
			AppID:       "de.volkswagen.car-net.eu.e-remote",
			AppName:     "We Connect",
			AppVersion:  "5.3.2",
			ClientBrand: "VW",
			ClientName:  "iPhone",
			Platform:    "iOS",
		}

		uri = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/register/v1"
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

		if err == nil {
			data := struct {
				ClientID string `json:"client_id"`
			}{}

			if err = v.DoJSON(req, &data); err == nil && data.ClientID == "" {
				err = errors.New("missing client id")
			}

			v.clientID = data.ClientID
		}

		if err == nil {
			data := url.Values(map[string][]string{
				"grant_type": {"id_token"},
				"scope":      {"sc2:fal"},
				"token":      {idToken},
			})

			req, err = request.New(http.MethodPost, vw.OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
				"Content-type": "application/x-www-form-urlencoded",
				"X-Client-Id":  v.clientID,
			})

			if err == nil {
				if err = v.DoJSON(req, &v.tokens); err == nil && v.tokens.AccessToken == "" {
					err = errors.New("missing access token")
				}
			}
		}
	}

	return err
}

func (v *VW) refreshHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-Client-Id":  v.clientID,
	}
}

// apiCall provides charger api response
func (v *VW) apiCall() (interface{}, error) {
	res, err := v.api.Charger()
	return res, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *VW) ChargeState() (float64, error) {
	res, err := v.apiG()
	if res, ok := res.(vw.ChargerResponse); err == nil && ok {
		return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), nil
	}

	return 0, err
}

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *VW) FinishTime() (time.Time, error) {
	res, err := v.apiG()
	if res, ok := res.(vw.ChargerResponse); err == nil && ok {
		var timestamp time.Time
		if err == nil {
			timestamp, err = time.Parse(time.RFC3339, res.Charger.Status.BatteryStatusData.RemainingChargingTime.Timestamp)
		}

		return timestamp.Add(time.Duration(res.Charger.Status.BatteryStatusData.RemainingChargingTime.Content) * time.Minute), err
	}

	return time.Time{}, err
}
