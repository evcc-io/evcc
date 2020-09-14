package vehicle

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"golang.org/x/net/publicsuffix"
)

const (
	vwIdentity = "https://identity.vwgroup.io"
	vwAPI      = "https://msg.volkswagen.de/fs-car"
	vwToken    = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"
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
	*util.HTTPHelper
	user, password, vin string
	brand, country      string
	tokens              audiTokenResponse
	chargeStateG        func() (float64, error)
	remainingTimeG      func() (time.Duration, error)
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
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(log),
		user:       cc.User,
		password:   cc.Password,
		vin:        cc.VIN,
		brand:      "Audi",
		country:    "DE",
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()
	v.remainingTimeG = provider.NewCached(v.remainingTime, cc.Cache).DurationGetter()

	var err error
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// track cookies and don't follow redirects
	v.Client = &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if err == nil {
		err = v.authFlow()
	}

	if err == nil && cc.VIN == "" {
		vehicles, err := v.vehicles()
		if err != nil {
			return nil, fmt.Errorf("cannot get vehicles: %v", err)
		}

		if len(vehicles) != 1 {
			return nil, fmt.Errorf("cannot find vehicle: %v", vehicles)
		}

		v.vin = vehicles[0]
		log.DEBUG.Printf("found vehicle: %v", v.vin)
	}

	return v, err
}

func (v *Audi) redirect(resp *http.Response, err error) (*http.Response, error) {
	if err == nil {
		uri := resp.Header.Get("Location")
		resp, err = v.Client.Get(uri)
	}

	return resp, err
}

func (v *Audi) request(method, uri string, data io.Reader, headers ...map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, uri, data)
	if err != nil {
		return req, err
	}

	for _, headers := range headers {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	return req, nil
}

type formVars struct {
	action     string
	csrf       string
	relayState string
	hmac       string
}

func formValues(reader io.Reader, id string) (formVars, error) {
	vars := formVars{}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err == nil {
		form := doc.Find(id).First()
		if form.Length() != 1 {
			return vars, errors.New("unexpected length")
		}

		var exists bool
		vars.action, exists = form.Attr("action")
		if !exists {
			return vars, errors.New("attribute not found")
		}

		vars.csrf, err = attr(form, "input[name=_csrf]", "value")
		if err == nil {
			vars.relayState, err = attr(form, "input[name=relayState]", "value")
		}
		if err == nil {
			vars.hmac, err = attr(form, "input[name=hmac]", "value")
		}
	}

	return vars, err
}

func attr(doc *goquery.Selection, path, attr string) (res string, err error) {
	sel := doc.Find(path)
	if sel.Length() != 1 {
		return "", errors.New("unexpected length")
	}

	v, exists := sel.Attr(attr)
	if !exists {
		return "", errors.New("attribute not found")
	}

	return v, nil
}

func (v *Audi) authFlow() error {
	var err error
	var uri, body string
	var vars formVars
	var req *http.Request
	var resp *http.Response

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?" +
		"response_type=code&client_id=09b6cbec-cd19-4589-82fd-363dfa8c24da%40apps_vw-dilab_com&" +
		"redirect_uri=myaudi%3A%2F%2F%2F&scope=address%20profile%20badge%20birthdate%20birthplace%20nationalIdentifier%20nationality%20profession%20email%20vin%20phone%20nickname%20name%20picture%20mbb%20gallery%20openid&" +
		"state=7f8260b5-682f-4db8-b171-50a5189a1c08&nonce=583b9af2-7799-4c72-9cb0-e6c0f42b87b3&prompt=login&ui_locales=de-DE"
	resp, err = v.Client.Get(uri)
	if err == nil {
		uri = resp.Header.Get("Location")
		resp, err = v.Client.Get(uri)
	}

	if err == nil {
		vars, err = formValues(resp.Body, "form#emailPasswordForm")
	}
	if err == nil {
		uri = vwIdentity + vars.action
		body := fmt.Sprintf(
			"_csrf=%s&relayState=%s&hmac=%s&email=%s",
			vars.csrf, vars.relayState, vars.hmac, url.QueryEscape(v.user),
		)
		req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(body))
	}
	if err == nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp, err = v.Client.Do(req)
	}

	if err == nil {
		uri = vwIdentity + resp.Header.Get("Location")
		req, err = http.NewRequest(http.MethodGet, uri, nil)

	}
	if err == nil {
		resp, err = v.Client.Do(req)
	}

	if err == nil {
		vars, err = formValues(resp.Body, "form#credentialsForm")
	}
	if err == nil {
		uri = vwIdentity + vars.action
		body = fmt.Sprintf(
			"_csrf=%s&relayState=%s&email=%s&hmac=%s&password=%s",
			vars.csrf,
			vars.relayState,
			url.QueryEscape(v.user),
			vars.hmac,
			url.QueryEscape(v.password),
		)
		req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(body))
	}
	if err == nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp, err = v.Client.Do(req)
	}

	for i := 6; i < 9; i++ {
		resp, err = v.redirect(resp, err)
	}

	var tokens audiTokenResponse
	if err == nil {
		var code string
		if location, err := url.Parse(resp.Header.Get("Location")); err == nil {
			code = location.Query().Get("code")
		}

		uri = "https://app-api.my.audi.com/myaudiappidk/v1/token"
		body = fmt.Sprintf(
			"client_id=%s&grant_type=%s&code=%s&redirect_uri=%s&response_type=%s",
			url.QueryEscape("09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"),
			"authorization_code",
			code,
			url.QueryEscape("myaudi:///"),
			url.QueryEscape("token id_token"),
		)

		req, err = v.request(http.MethodPost, uri, strings.NewReader(body),
			map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		)
	}
	if err == nil {
		_, err = v.RequestJSON(req, &tokens)
	}

	if err == nil {
		body = fmt.Sprintf("grant_type=%s&token=%s&scope=%s", "id_token", tokens.IDToken, "sc2:fal")
		headers := map[string]string{
			"Content-Type":  "application/x-www-form-urlencoded",
			"X-App-Version": "3.14.0",
			"X-App-Name":    "myAudi",
			"X-Client-Id":   "77869e21-e30a-4a92-b016-48ab7d3db1d8",
		}

		req, err = v.request(http.MethodPost, vwToken, strings.NewReader(body), headers)
	}
	if err == nil {
		_, err = v.RequestJSON(req, &tokens)
		v.tokens = tokens
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

	req, err := v.request(http.MethodPost, vwToken, strings.NewReader(body), headers)
	if err == nil {
		var tokens audiTokenResponse
		_, err = v.RequestJSON(req, &tokens)
		if err == nil {
			v.tokens = tokens
		}
	}

	return err
}

func (v *Audi) getJSON(uri string, res interface{}) error {
	req, err := v.request(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.tokens.AccessToken,
	})

	if err == nil {
		_, err = v.RequestJSON(req, &res)

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
				_, err = v.RequestJSON(req, &res)
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

// remainingTime implements the Vehicle.ChargeRemainder interface
func (v *Audi) remainingTime() (time.Duration, error) {
	var res audiChargerResponse
	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", vwAPI, v.brand, v.country, v.vin)
	err := v.getJSON(uri, &res)

	var timestamp time.Time
	if err == nil {
		timestamp, err = time.Parse(time.RFC3339, res.Charger.Status.BatteryStatusData.RemainingChargingTime.Timestamp)
	}

	return time.Duration(res.Charger.Status.BatteryStatusData.RemainingChargingTime.Content)*time.Minute - time.Since(timestamp), err
}

// RemainingTime implements the Vehicle.ChargeRemainder interface
func (v *Audi) RemainingTime() (time.Duration, error) {
	return v.remainingTimeG()
}
