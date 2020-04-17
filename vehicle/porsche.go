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
)

const (
	porscheAuth = "https://login.porsche.com/as/authorization.oauth2"
	porscheAPI  = "https://www.bmw-connecteddrive.com/api"
)

type porscheDynamicResponse struct {
	AttributesMap struct {
		ChargingLevelHv float64 `json:"chargingLevelHv,string"`
	}
}

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	*api.HTTPHelper
	user, password, vin string
	token               string
	tokenValid          time.Time
	chargeStateG        provider.FloatGetter
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

	v := &Porsche{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: api.NewHTTPHelper(api.NewLogger("porsche ")),
		user:       cc.User,
		password:   cc.Password,
		vin:        cc.VIN,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v
}

func (v *Porsche) login(user, password string) error {
	data := url.Values{
		"username":      []string{user},
		"password":      []string{password},
		"client_id":     []string{"TZ4Vf5wnKeipJxvatJ60lPHYEzqZ4WNp"},
//		"redirect_uri":  []string{"https://www.bmw-connecteddrive.com/app/default/static/external-dispatch.html"},
		"response_type": []string{"token"},
		"scope":         []string{"authenticate_user fupo"},
//		"state":         []string{"eyJtYXJrZXQiOiJkZSIsImxhbmd1YWdlIjoiZGUiLCJkZXN0aW5hdGlvbiI6ImxhbmRpbmdQYWdlIn0"},
		"locale":        []string{"DE-de"},
	}

	req, err := http.NewRequest(http.MethodPost, porscheAuth, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }, // don't follow redirects
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	query, err := url.ParseQuery(resp.Header.Get("Location"))
	if err != nil {
		return err
	}

	token := query.Get("access_token")
	expires, err := strconv.Atoi(query.Get("expires_in"))
	if err != nil || token == "" || expires == 0 {
		return errors.New("could not obtain token")
	}

	v.token = token
	v.tokenValid = time.Now().Add(time.Duration(expires) * time.Second)

	return nil
}

func (v *Porsche) request(uri string) (*http.Request, error) {
	if v.token == "" || time.Since(v.tokenValid) > 0 {
		if err := v.login(v.user, v.password); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v.token))
	}

	return req, nil
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Porsche) chargeState() (float64, error) {
	uri := fmt.Sprintf("%s/vehicle/dynamic/v1/%s", porscheAPI, v.vin)
	req, err := v.request(uri)
	if err != nil {
		return 0, err
	}

	var br porscheDynamicResponse
	_, err = v.RequestJSON(req, &br)

	return br.AttributesMap.ChargingLevelHv, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Porsche) ChargeState() (float64, error) {
	return v.chargeStateG()
}
