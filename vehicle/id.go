package vehicle

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/oidc"
	"github.com/andig/evcc/vehicle/vw"
	"golang.org/x/net/publicsuffix"
)

// ID is an api.Vehicle implementation for VW ID cars
type ID struct {
	*embed
	*request.Helper
	user, password string
	vin            string
	userInfo       UserInfo
	csrf           string
	carTokens      oidc.Tokens
	weTokens       oidc.Tokens
	chargerG       func() (interface{}, error)
}

// UserInfo is the https://www.volkswagen.de/app/authproxy/vw-de/user response
type UserInfo struct {
	Sub           string
	Name          string
	GivenName     string
	FamilyName    string
	Email         string
	EmailVerified bool
	UpdatedAt     int64
}

func init() {
	registry.Add("id", NewIDFromConfig)
}

// NewIDFromConfig creates a new vehicle
func NewIDFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("id")

	v := &ID{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	var err error
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// track cookies and follow all redirects
	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return nil
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

	v.chargerG = provider.NewCached(v.chargeState, cc.Cache).InterfaceGetter()

	return v, err
}

func (v *ID) authFlow() error {
	var uri string
	var req *http.Request

	uri = "https://www.volkswagen.de/app/authproxy/login?fag=vw-de,vwag-weconnect&scope-vw-de=profile,address,phone,carConfigurations,dealers,cars,vin,profession&scope-vwag-weconnect=openid&prompt-vw-de=login&prompt-vwag-weconnect=none&redirectUrl=https://www.volkswagen.de/de/besitzer-und-nutzer/myvolkswagen.html"
	resp, err := v.Get(uri)

	var vars vw.FormVars
	if err == nil {
		vars, err = vw.FormValues(resp.Body, "form#emailPasswordForm")
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/identifier
	if err == nil {
		data := url.Values(map[string][]string{
			"_csrf":      {vars.Csrf},
			"relayState": {vars.RelayState},
			"hmac":       {vars.Hmac},
			"email":      {v.user},
		})

		uri = vw.IdentityURI + vars.Action
		if resp, err = v.PostForm(uri, data); err == nil {
			vars, err = vw.FormValues(resp.Body, "form#credentialsForm")
		}
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/authenticate
	if err == nil {
		data := url.Values(map[string][]string{
			"_csrf":      {vars.Csrf},
			"relayState": {vars.RelayState},
			"hmac":       {vars.Hmac},
			"email":      {v.user},
			"password":   {v.password},
		})

		uri = vw.IdentityURI + vars.Action
		if _, err = v.PostForm(uri, data); err == nil {
			vwDomain, _ := url.Parse("https://www.volkswagen.de/")
			cookies := v.Client.Jar.Cookies(vwDomain)

			for _, c := range cookies {
				if c.Name == "csrf_token" {
					v.csrf = c.Value
					break
				}
			}

			if v.csrf == "" {
				err = errors.New("missing csrf token")
			}
		}
	}

	if err == nil {
		req, err = request.New(http.MethodGet, "https://www.volkswagen.de/app/authproxy/vw-de/user", nil, map[string]string{
			"Accept":       "application/json",
			"X-csrf-token": v.csrf,
		})

		if err == nil {
			err = v.DoJSON(req, &v.userInfo)
		}
	}

	if err == nil {
		uri = "https://www.volkswagen.de/app/authproxy/vw-de/tokens"
		req, err = request.New(http.MethodGet, uri, nil, map[string]string{
			"Accept":       "application/json",
			"X-csrf-token": v.csrf,
		})

		if err == nil {
			if err = v.DoJSON(req, &v.carTokens); err == nil {
				if v.carTokens.AccessToken == "" {
					err = errors.New("missing vw-de access token")
				}
			}
		}
	}

	if err == nil {
		uri = "https://www.volkswagen.de/app/authproxy/vwag-weconnect/tokens"
		req, err = request.New(http.MethodGet, uri, nil, map[string]string{
			"Accept":       "application/json",
			"X-csrf-token": v.csrf,
		})

		if err == nil {
			if err = v.DoJSON(req, &v.weTokens); err == nil {
				if v.weTokens.AccessToken == "" {
					err = errors.New("missing vwag-weconnect access token")
				}
			}
		}

		if err == nil {
			uri = "https://myvw-idk-token-exchanger.apps.emea.vwapps.io/token-exchange?isWcar=true"
			req, err = request.New(http.MethodGet, uri, nil, map[string]string{
				"Accept":        "applicaton/json, text/plain",
				"Authorization": "Bearer " + v.weTokens.AccessToken,
			})

			if err == nil {
				if resp, err = v.Do(req); err == nil {
					var body []byte
					if body, err = request.ReadBody(resp); err == nil {
						v.weTokens.AccessToken = string(body)
					}
				}
			}
		}
	}

	return err
}

func (v *ID) vehicles() (res []string, err error) {
	uri := "https://w1hub-backend-production.apps.emea.vwapps.io/cars"
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.carTokens.AccessToken,
	})

	var vehicles []struct {
		VIN string
	}

	if err == nil {
		err = v.DoJSON(req, &vehicles)

		for _, v := range vehicles {
			res = append(res, v.VIN)
		}
	}

	return res, err
}

type idData struct {
	Error struct {
		Code    int
		Message string
	}
	Data []struct {
		ID                   string
		CarCapturedTimestamp string
		Properties           []struct {
			Name, Value string // engineType: electric, remainingRange_km, currentSOC_pct
		}
	}
}

func (v *ID) chargeState() (interface{}, error) {
	uri := fmt.Sprintf("https://cardata.apps.emea.vwapps.io/vehicles/%s/fuel/status", v.vin)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.weTokens.AccessToken,
		"User-Id":       v.userInfo.Sub,
	})

	var state idData
	if err == nil {
		err = v.DoJSON(req, &state)

		if err != nil {
			// handle http 401, 403
			if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusUnauthorized, http.StatusForbidden) {
				if err = v.authFlow(); err == nil {
					// re-do request with new token
					req.Header.Set("Authorization", "Bearer "+v.weTokens.AccessToken)
					err = v.DoJSON(req, &state)
				}
			}
		}
	}

	return state, err
}

func (v *ID) extractProperty(data idData, property string) (int64, error) {
	for _, d := range data.Data {
		for _, p := range d.Properties {
			if p.Name == property {
				i, err := strconv.Atoi(p.Value)
				return int64(i), err
			}
		}
	}

	return 0, fmt.Errorf("missing %s", property)
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *ID) ChargeState() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(idData); err == nil && ok {
		var i int64
		if i, err = v.extractProperty(res, "currentSOC_pct"); err == nil {
			return float64(i), nil
		}
	}

	return 0, err
}

// Range implements the Vehicle.Range interface
func (v *ID) Range() (int64, error) {
	res, err := v.chargerG()
	if res, ok := res.(idData); err == nil && ok {
		var i int64
		if i, err = v.extractProperty(res, "remainingRange_km"); err == nil {
			return i, nil
		}
	}

	return 0, err
}
