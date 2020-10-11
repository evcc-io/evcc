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

// https://github.com/davidgiga1993/AudiAPI

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*request.Helper
	user, password string
	tokens         vw.Tokens
	api            *vw.API
	apiG           func() (interface{}, error)
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

const audiClientID = "77869e21-e30a-4a92-b016-48ab7d3db1d8"

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
	}

	v.api = vw.NewAPI(v.Helper, &v.tokens, v.authFlow, v.refreshHeaders, strings.ToUpper(cc.VIN), "Audi", "DE")
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

func (v *Audi) authFlow() error {
	var err error
	var uri string
	var req *http.Request
	var resp *http.Response
	var tokens vw.Tokens

	const clientID = "09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"
	const redirectURI = "myaudi:///"

	query := url.Values(map[string][]string{
		"response_type": {"code"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"scope":         {"address profile badge birthdate birthplace nationalIdentifier nationality profession email vin phone nickname name picture mbb gallery openid"},
		"state":         {"7f8260b5-682f-4db8-b171-50a5189a1c08"},
		"nonce":         {"7f8260b5-682f-4db8-b171-50a5189a1c08"},
		"prompt":        {"login"},
		"ui_locales":    {"de-DE"},
	})

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?" + query.Encode()
	if err == nil {
		identity := &vw.Identity{Client: v.Client}
		resp, err = identity.Login(uri, v.user, v.password)
	}

	if err == nil {
		var code string
		if location, err := url.Parse(resp.Header.Get("Location")); err == nil {
			code = location.Query().Get("code")
		}

		data := url.Values(map[string][]string{
			"client_id":     {clientID},
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {redirectURI},
			"response_type": {"token id_token"},
		})

		uri = "https://app-api.my.audi.com/myaudiappidk/v1/token"
		req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
		if err == nil {
			if err = v.DoJSON(req, &tokens); err == nil && tokens.IDToken == "" {
				err = errors.New("missing id token (1)")
			}
		}
	}

	if err == nil {
		data := url.Values(map[string][]string{
			"grant_type": {"id_token"},
			"scope":      {"sc2:fal"},
			"token":      {tokens.IDToken},
		})

		req, err = request.New(http.MethodPost, vw.OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"X-Client-Id":  audiClientID,
		})
		if err == nil {
			if err = v.DoJSON(req, &v.tokens); err == nil && tokens.IDToken == "" {
				err = errors.New("missing id token (2)")
			}
		}
	}

	return err
}

func (v *Audi) refreshHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"X-App-Version": "3.14.0",
		"X-App-Name":    "myAudi",
		"X-Client-Id":   audiClientID,
	}
}

// apiCall provides charger api response
func (v *Audi) apiCall() (interface{}, error) {
	res, err := v.api.Charger()
	return res, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Audi) ChargeState() (float64, error) {
	res, err := v.apiG()
	if res, ok := res.(vw.ChargerResponse); err == nil && ok {
		return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), nil
	}

	return 0, err
}

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *Audi) FinishTime() (time.Time, error) {
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
