package vehicle

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"golang.org/x/net/publicsuffix"
)

type vwTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type vwVehiclesResponse struct {
	UserVehicles struct {
		Vehicle []string
	}
}

type vwChargerResponse struct {
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

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	*util.HTTPHelper
	user, password, vin string
	brand, country      string
	tokens              vwTokenResponse
	chargeStateG        func() (float64, error)
	finishTimeG         func() (time.Time, error)
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

	log := util.NewLogger("audi")

	v := &VW{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(log),
		user:       cc.User,
		password:   cc.Password,
		vin:        cc.VIN,
		brand:      "Audi",
		country:    "DE",
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()
	v.finishTimeG = provider.NewCached(v.finishTime, cc.Cache).TimeGetter()

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
		Transport: v,
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

func (v *VW) RoundTrip(req *http.Request) (*http.Response, error) {
	b, err := httputil.DumpRequest(req, true)
	if err == nil {
		v.HTTPHelper.Log.TRACE.Println("\n" + string(b))
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		b, err := httputil.DumpResponse(resp, false)
		if err == nil {
			v.HTTPHelper.Log.TRACE.Println("\n" + string(b))
		}
	}

	return resp, err
}

func (v *VW) redirect(resp *http.Response, err error) (*http.Response, error) {
	if err == nil {
		uri := resp.Header.Get("Location")
		resp, err = v.Client.Get(uri)
	}

	return resp, err
}

func (v *VW) request(method, uri string, data io.Reader, headers ...map[string]string) (*http.Request, error) {
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

func (v *VW) authFlow() error {
	var err error
	var uri, body string
	var vars formVars
	var req *http.Request
	var resp *http.Response

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?" +
		"response_type=code&client_id=09b6cbec-cd19-4589-82fd-363dfa8c24da%40apps_vw-dilab_com&" +
		"redirect_uri=myaudi%3A%2F%2F%2F&scope=address%20profile%20badge%20birthdate%20birthplace%20nationalIdentifier%20nationality%20profession%20email%20vin%20phone%20nickname%20name%20picture%20mbb%20gallery%20openid&" +
		"state=7f8260b5-682f-4db8-b171-50a5189a1c08&nonce=583b9af2-7799-4c72-9cb0-e6c0f42b87b3&prompt=login&ui_locales=de-DE"

	// vw only
	uri = "https://www.portal.volkswagen-we.com/portal/en_GB/web/guest/home"
	resp, err = v.Client.Get(uri)

	uri = "https://identity.vwgroup.io/oidc/v1/authorize?" +
		"ui_locales=de&scope=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&" +
		"response_type=code&state=gmiJOaB4&" +
		"redirect_uri=https%3A%2F%2Fwww.portal.volkswagen-we.com%2Fportal%2Fweb%2Fguest%2Fcomplete-login&nonce=38042ee3-b7a7-43cf-a9c1-63d2f3f2d9f3&prompt=login&client_id=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com"

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

	var tokens vwTokenResponse
	if err == nil {
		var code, state string
		var locationURL *url.URL
		location := resp.Header.Get("Location")
		locationURL, err = url.Parse(location)
		if err == nil {
			code = locationURL.Query().Get("code")
			state = locationURL.Query().Get("state")
		}

		// resp, err = v.redirect(resp, err)

		if strings.Contains(location, "complete-login") {
			_ = state
			_ = code

			ref := location
			uri = fmt.Sprintf(
				"%s?p_auth=%s&p_p_id=33_WAR_cored5portlet&p_p_lifecycle=1&p_p_state=normal&p_p_mode=view&p_p_col_id=column-1&p_p_col_count=1&_33_WAR_cored5portlet_javax.portlet.action=getLoginStatus",
				locationURL.Scheme+"://"+locationURL.Host+locationURL.Path,
				state,
			)
			body = fmt.Sprintf("_33_WAR_cored5portlet_code=%s", code)
			req, err = v.request(http.MethodPost, uri, strings.NewReader(body),
				map[string]string{
					"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:68.0) Gecko/20100101 Firefox/68.0",
					"Referer":    ref,
					// "Accept-Encoding": "gzip, deflate, br",
					"Accept":          "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,*/*;q=0.8",
					"Accept-Language": "en-US,nl;q=0.7,en;q=0.3",
					"Content-Type":    "application/x-www-form-urlencoded",
					"Content-Length":  strconv.Itoa(len(body)),
				},
			)
		}

		if err == nil {
			resp, err = v.Client.Do(req)
			time.Sleep(time.Second)
			os.Exit(1)
		}

		// uri = "https://app-api.my.audi.com/myaudiappidk/v1/token"
		// body = fmt.Sprintf(
		// 	"client_id=%s&grant_type=%s&code=%s&redirect_uri=%s&response_type=%s",
		// 	url.QueryEscape("09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"),
		// 	"authorization_code",
		// 	code,
		// 	url.QueryEscape("myaudi:///"),
		// 	url.QueryEscape("token id_token"),
		// )

		// req, err = v.request(http.MethodPost, uri, strings.NewReader(body),
		// 	map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		// )
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

func (v *VW) refreshToken() error {
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
		var tokens vwTokenResponse
		_, err = v.RequestJSON(req, &tokens)
		if err == nil {
			v.tokens = tokens
		}
	}

	return err
}

func (v *VW) getJSON(uri string, res interface{}) error {
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

func (v *VW) vehicles() ([]string, error) {
	var res vwVehiclesResponse
	uri := fmt.Sprintf("%s/usermanagement/users/v1/Audi/DE/vehicles", vwAPI)
	err := v.getJSON(uri, &res)
	return res.UserVehicles.Vehicle, err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *VW) chargeState() (float64, error) {
	var res vwChargerResponse
	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", vwAPI, v.brand, v.country, v.vin)
	err := v.getJSON(uri, &res)
	return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *VW) ChargeState() (float64, error) {
	return v.chargeStateG()
}

// finishTime implements the Vehicle.ChargeFinishTimer interface
func (v *VW) finishTime() (time.Time, error) {
	var res vwChargerResponse
	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", vwAPI, v.brand, v.country, v.vin)
	err := v.getJSON(uri, &res)

	var timestamp time.Time
	if err == nil {
		timestamp, err = time.Parse(time.RFC3339, res.Charger.Status.BatteryStatusData.RemainingChargingTime.Timestamp)
	}

	return timestamp.Add(time.Duration(res.Charger.Status.BatteryStatusData.RemainingChargingTime.Content) * time.Minute), err
}

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *VW) FinishTime() (time.Time, error) {
	return v.finishTimeG()
}
