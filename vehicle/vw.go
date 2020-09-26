package vehicle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/vwidentity"
	"golang.org/x/net/publicsuffix"
)

type vwVehiclesResponse struct {
	FullyLoadedVehiclesResponse struct {
		CompleteVehicles []struct {
			VIN string
		}
		VehiclesNotFullyLoaded []struct {
			VIN string
		}
	}
}

type vwChargerResponse struct {
	ErrorCode         int `json:",string"`
	VehicleStatusData struct {
		BatteryRange int
		BatteryLevel int
	}
}

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	*util.HTTPHelper
	user, password, vin string
	baseURI, csrf       string
	identity            *vwidentity.Identity
	chargeStateG        func() (float64, error)
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
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

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
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, err
}

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
	var uri, body string
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
		uri = "https://www.portal.volkswagen-we.com/portal/en_GB/web/guest/home/-/csrftokenhandling/get-login-url"
		if req, err = vwidentity.Request(http.MethodPost, uri, nil, map[string]string{"X-CSRF-Token": vars.Csrf}); err == nil {
			resp, err = v.Client.Do(req)
		}
	}

	// get login url
	if err == nil {
		uri, err = v.loginURL(resp)
		uri = strings.ReplaceAll(uri, " ", "%20")

		if err == nil {
			resp, err = v.identity.Login(uri, v.user, v.password)
		}
	}

	// get base url
	if err == nil {
		var code, state string
		var locationURL *url.URL

		location := resp.Header.Get("Location")

		locationURL, err = url.Parse(location)
		if err == nil {
			code = locationURL.Query().Get("code")
			state = locationURL.Query().Get("state")
		}

		uri = fmt.Sprintf(
			"%s?p_auth=%s&p_p_id=33_WAR_cored5portlet&p_p_lifecycle=1&p_p_state=normal&p_p_mode=view&p_p_col_id=column-1&p_p_col_count=1&_33_WAR_cored5portlet_javax.portlet.action=getLoginStatus",
			locationURL.Scheme+"://"+locationURL.Host+locationURL.Path,
			state,
		)

		body = fmt.Sprintf("_33_WAR_cored5portlet_code=%s", url.QueryEscape(code))

		req, err = vwidentity.Request(http.MethodPost, uri, strings.NewReader(body),
			map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		)

		if err == nil {
			resp, err = v.Client.Do(req)
			uri = resp.Header.Get("Location")

			v.baseURI = uri
			v.csrf = vars.Csrf
		}
	}

	return err
}

func (v *VW) vehicles() ([]string, error) {
	uri := v.baseURI + "/-/mainnavigation/get-fully-loaded-cars"

	req, err := vwidentity.Request(http.MethodPost, uri, nil,
		map[string]string{"Accept": "application/json", "X-CSRF-Token": v.csrf},
	)

	var res vwVehiclesResponse
	if err == nil {
		_, err = v.RequestJSON(req, &res)
	}

	vehicles := make([]string, 0)
	for _, v := range res.FullyLoadedVehiclesResponse.CompleteVehicles {
		vehicles = append(vehicles, v.VIN)
	}
	for _, v := range res.FullyLoadedVehiclesResponse.VehiclesNotFullyLoaded {
		vehicles = append(vehicles, v.VIN)
	}

	return vehicles, err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *VW) chargeState() (float64, error) {
	var res vwChargerResponse

	uri := v.baseURI + "/-/vsr/get-vsr"

	req, err := vwidentity.Request(http.MethodPost, uri, nil,
		map[string]string{"Accept": "application/json", "X-CSRF-Token": v.csrf},
	)

	if err == nil {
		_, err = v.RequestJSON(req, &res)
	}

	return float64(res.VehicleStatusData.BatteryLevel), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *VW) ChargeState() (float64, error) {
	return v.chargeStateG()
}
