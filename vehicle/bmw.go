package vehicle

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/oauth"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	bmwAuth = "https://customer.bmwgroup.com/gcdm/oauth/authenticate"
	bmwAPI  = "https://www.bmw-connecteddrive.com/api"
)

type bmwDynamicResponse struct {
	AttributesMap struct {
		ChargingLevelHv float64 `json:"chargingLevelHv,string"`
	}
}

type bmwVehiclesResponse []struct {
	VIN string `json:"vin"`
}

// BMW is an api.Vehicle implementation for BMW cars
type BMW struct {
	*embed
	*request.Helper
	user, password, vin string
	chargeStateG        func() (float64, error)
}

func init() {
	registry.Add("bmw", NewBMWFromConfig)
}

// NewBMWFromConfig creates a new vehicle
func NewBMWFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("bmw")

	v := &BMW{
		embed:    &cc.embed,
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	token, err := v.RefreshToken(nil)
	if err != nil {
		return nil, err
	}

	v.Client.Transport = &oauth2.Transport{
		Source: oauth.RefreshTokenSource(token, v),
		Base:   v.Client.Transport,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	if cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, err
}

func (v *BMW) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"username":      []string{v.user},
		"password":      []string{v.password},
		"client_id":     []string{"dbf0a542-ebd1-4ff0-a9a7-55172fbfce35"},
		"redirect_uri":  []string{"https://www.bmw-connecteddrive.com/app/default/static/external-dispatch.html"},
		"response_type": []string{"token"},
		"scope":         []string{"authenticate_user fupo"},
		"state":         []string{"eyJtYXJrZXQiOiJkZSIsImxhbmd1YWdlIjoiZGUiLCJkZXN0aW5hdGlvbiI6ImxhbmRpbmdQYWdlIn0"},
		"locale":        []string{"DE-de"},
	}

	req, err := request.New(http.MethodPost, bmwAuth, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	// don't follow redirects
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	defer func() { v.Client.CheckRedirect = nil }()

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}

	query, err := url.ParseQuery(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	at := query.Get("access_token")
	expires, err := strconv.Atoi(query.Get("expires_in"))
	if err != nil || at == "" || expires == 0 {
		return nil, errors.New("could not obtain token")
	}

	token := &oauth2.Token{
		AccessToken: at,
		Expiry:      time.Now().Add(time.Duration(expires) * time.Second),
	}

	return token, nil
}

// vehicles implements returns the list of user vehicles
func (v *BMW) vehicles() ([]string, error) {
	var resp bmwVehiclesResponse
	uri := fmt.Sprintf("%s/me/vehicles/v2/", bmwAPI)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	var vehicles []string
	for _, v := range resp {
		vehicles = append(vehicles, v.VIN)
	}

	return vehicles, err
}

// chargeState implements the api.Vehicle interface
func (v *BMW) chargeState() (float64, error) {
	var resp bmwDynamicResponse
	uri := fmt.Sprintf("%s/vehicle/dynamic/v1/%s", bmwAPI, v.vin)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return 0, err
	}

	err = v.DoJSON(req, &resp)
	return resp.AttributesMap.ChargingLevelHv, err
}

// SoC implements the api.Vehicle interface
func (v *BMW) SoC() (float64, error) {
	return v.chargeStateG()
}
