package vehicle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/vwidentity"
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
	identity            *vwidentity.Identity
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

	// v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()
	// v.finishTimeG = provider.NewCached(v.finishTime, cc.Cache).TimeGetter()

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

	// if err == nil && cc.VIN == "" {
	// 	v.vin, err = findVehicle(v.vehicles())
	// 	if err == nil {
	// 		log.DEBUG.Printf("found vehicle: %v", v.vin)
	// 	}
	// }

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

// func (v *VW) dumpBody(resp *http.Response) {
// 	b, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer resp.Body.Close()
// 	v.HTTPHelper.Log.TRACE.Println(string(b))
// 	panic("foo")
// }

func (v *VW) loginURL(resp *http.Response) (string, error) {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	res := struct {
		ErrorCode int `json:",string"`
		LoginURL  struct {
			Path string
		}
	}{}

	err = json.Unmarshal(b, &res)
	if err == nil && res.ErrorCode != 0 {
		err = fmt.Errorf("login url error code: %d", res.ErrorCode)
	}

	return res.LoginURL.Path, err
}

func (v *VW) authFlow() error {
	var err error
	var uri, ref, body string
	var vars vwidentity.FormVars
	var req *http.Request
	var resp *http.Response

	v.identity = &vwidentity.Identity{Client: v.Client}

	// GET www.portal.volkswagen-we.com/portal/de_DE/web/guest/home
	uri = "https://www.portal.volkswagen-we.com/portal/de_DE/web/guest/home"
	resp, err = v.Client.Get(uri)

	if err == nil {
		vars, err = vwidentity.FormValues(resp.Body, "meta")
	}

	// POST www.portal.volkswagen-we.com/portal/en_GB/web/guest/home/-/csrftokenhandling/get-login-url
	if err == nil {
		ref = uri

		uri = "https://www.portal.volkswagen-we.com/portal/en_GB/web/guest/home/-/csrftokenhandling/get-login-url"
		req, err = vwidentity.Request(http.MethodPost, uri, nil,
			map[string]string{
				"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:68.0) Gecko/20100101 Firefox/68.0",
				"Referer":         ref,
				"Accept":          "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,*/*;q=0.8",
				"Accept-Language": "en-US,nl;q=0.7,en;q=0.3",
				"X-CSRF-Token":    vars.Csrf,
			},
		)
		if err == nil {
			resp, err = v.Client.Do(req)
		}
	}

	// get login url
	if err == nil {
		uri, err = v.loginURL(resp)
		uri = strings.ReplaceAll(uri, " ", "%20")
	}

	if err == nil {
		resp, err = v.identity.Login(uri, v.user, v.password)
	}

	// var tokens vwTokenResponse
	if err == nil {
		var code, state string
		var locationURL *url.URL
		location := resp.Header.Get("Location")
		locationURL, err = url.Parse(location)
		if err == nil {
			code = locationURL.Query().Get("code")
			state = locationURL.Query().Get("state")
		}

		if strings.Contains(location, "complete-login") {
			_ = state
			_ = code

			ref := location
			_ = ref
			uri = fmt.Sprintf(
				"%s?p_auth=%s&p_p_id=33_WAR_cored5portlet&p_p_lifecycle=1&p_p_state=normal&p_p_mode=view&p_p_col_id=column-1&p_p_col_count=1&_33_WAR_cored5portlet_javax.portlet.action=getLoginStatus",
				locationURL.Scheme+"://"+locationURL.Host+locationURL.Path,
				state,
			)

			body = fmt.Sprintf("_33_WAR_cored5portlet_code=%s", url.QueryEscape(code))

			req, err = vwidentity.Request(http.MethodPost, uri, strings.NewReader(body),
				map[string]string{
					"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:68.0) Gecko/20100101 Firefox/68.0",
					"Referer":         ref,
					"Accept":          "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,*/*;q=0.8",
					"Accept-Language": "en-US,nl;q=0.7,en;q=0.3",
					"Content-Type":    "application/x-www-form-urlencoded",
					"Content-Length":  strconv.Itoa(len(body)),
				},
			)
		}

		if err == nil {
			resp, err = v.Client.Do(req)
			uri = resp.Header.Get("Location")
		}

		if err == nil {
			// html
			req, err = vwidentity.Request(http.MethodGet, uri, nil,
				map[string]string{
					"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:68.0) Gecko/20100101 Firefox/68.0",
					"Referer":         ref,
					"Accept":          "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,*/*;q=0.8",
					"Accept-Language": "en-US,nl;q=0.7,en;q=0.3",
					"X-CSRF-Token":    csrf,
				},
			)
			resp, err = v.Client.Do(req)

			// if err == nil {
			// 	println(csrf)
			// 	vars, err = FormValues(resp.Body, "meta")
			// 	csrf := vars.Csrf
			// 	_ = csrf
			// }

			if err == nil {
				uri += "/-/mainnavigation/get-fully-loaded-cars"

				req, err = vwidentity.Request(http.MethodGet, uri, nil,
					map[string]string{
						"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:68.0) Gecko/20100101 Firefox/68.0",
						"Referer":         ref,
						"Accept":          "application/json, text/plain, */*",
						"Accept-Language": "en-US,nl;q=0.7,en;q=0.3",
						"X-CSRF-Token":    csrf,
					},
				)
				resp, err = v.Client.Do(req)

				// v.dumpBody(resp)
			}

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

		// req, err = vwidentity.Request(http.MethodPost, uri, strings.NewReader(body),
		// 	map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		// )
	}
	// if err == nil {
	// 	_, err = vwidentity.RequestJSON(req, &tokens)
	// }

	// if err == nil {
	// 	body = fmt.Sprintf("grant_type=%s&token=%s&scope=%s", "id_token", tokens.IDToken, "sc2:fal")
	// 	headers := map[string]string{
	// 		"Content-Type":  "application/x-www-form-urlencoded",
	// 		"X-App-Version": "3.14.0",
	// 		"X-App-Name":    "myAudi",
	// 		"X-Client-Id":   "77869e21-e30a-4a92-b016-48ab7d3db1d8",
	// 	}

	// 	req, err = vwidentity.Request(http.MethodPost, vwToken, strings.NewReader(body), headers)
	// }
	// if err == nil {
	// 	_, err = vwidentity.RequestJSON(req, &tokens)
	// 	v.tokens = tokens
	// }

	return err
}

// func (v *VW) refreshToken() error {
// 	if v.tokens.RefreshToken == "" {
// 		return errors.New("missing refresh token")
// 	}

// 	body := fmt.Sprintf("grant_type=%s&refresh_token=%s&scope=%s", "refresh_token", v.tokens.RefreshToken, "sc2:fal")
// 	headers := map[string]string{
// 		"Content-Type":  "application/x-www-form-urlencoded",
// 		"X-App-Version": "3.14.0",
// 		"X-App-Name":    "myAudi",
// 		"X-Client-Id":   "77869e21-e30a-4a92-b016-48ab7d3db1d8",
// 	}

// 	req, err := vwidentity.Request(http.MethodPost, vwToken, strings.NewReader(body), headers)
// 	if err == nil {
// 		var tokens vwTokenResponse
// 		_, err = vwidentity.RequestJSON(req, &tokens)
// 		if err == nil {
// 			v.tokens = tokens
// 		}
// 	}

// 	return err
// }

// func (v *VW) getJSON(uri string, res interface{}) error {
// 	req, err := vwidentity.Request(http.MethodGet, uri, nil, map[string]string{
// 		"Accept":        "application/json",
// 		"Authorization": "Bearer " + v.tokens.AccessToken,
// 	})

// 	if err == nil {
// 		_, err = vwidentity.RequestJSON(req, &res)

// 		// token expired?
// 		if err != nil {
// 			resp := v.LastResponse()

// 			// handle http 401
// 			if resp != nil && resp.StatusCode == http.StatusUnauthorized {
// 				// use refresh token
// 				err = v.refreshToken()

// 				// re-run auth flow
// 				if err != nil {
// 					err = v.authFlow()
// 				}
// 			}

// 			// retry original requests
// 			if err == nil {
// 				req.Header.Set("Authorization", "Bearer "+v.tokens.AccessToken)
// 				_, err = vwidentity.RequestJSON(req, &res)
// 			}
// 		}
// 	}

// 	return err
// }

// func (v *VW) vehicles() ([]string, error) {
// 	var res vwVehiclesResponse
// 	uri := fmt.Sprintf("%s/usermanagement/users/v1/Audi/DE/vehicles", vwAPI)
// 	err := v.getJSON(uri, &res)
// 	return res.UserVehicles.Vehicle, err
// }

// // chargeState implements the Vehicle.ChargeState interface
// func (v *VW) chargeState() (float64, error) {
// 	var res vwChargerResponse
// 	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", vwAPI, v.brand, v.country, v.vin)
// 	err := v.getJSON(uri, &res)
// 	return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), err
// }

// ChargeState implements the Vehicle.ChargeState interface
func (v *VW) ChargeState() (float64, error) {
	// return v.chargeStateG()
	return 0, nil
}

// // finishTime implements the Vehicle.ChargeFinishTimer interface
// func (v *VW) finishTime() (time.Time, error) {
// 	var res vwChargerResponse
// 	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", vwAPI, v.brand, v.country, v.vin)
// 	err := v.getJSON(uri, &res)

// 	var timestamp time.Time
// 	if err == nil {
// 		timestamp, err = time.Parse(time.RFC3339, res.Charger.Status.BatteryStatusData.RemainingChargingTime.Timestamp)
// 	}

// 	return timestamp.Add(time.Duration(res.Charger.Status.BatteryStatusData.RemainingChargingTime.Content) * time.Minute), err
// }

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *VW) FinishTime() (time.Time, error) {
	// return v.finishTimeG()
	return time.Time{}, nil
}
