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
	"github.com/andig/evcc/util/request"
)

const (
	fordAuth = "https://fcis.ice.ibmcloud.com"
	fordAPI  = "https://usapi.cv.ford.com"
)

// Ford is an api.Vehicle implementation for Ford cars
type Ford struct {
	*embed
	*request.Helper
	user, password, vin string
	token               string
	tokenValid          time.Time
	chargeStateG        func() (float64, error)
}

func init() {
	registry.Add("ford", NewFordFromConfig)
}

// NewFordFromConfig creates a new vehicle
func NewFordFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ford")

	v := &Ford{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	var err error
	if cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, nil
}

func (v *Ford) login(user, password string) error {
	data := url.Values{
		"client_id":  []string{"9fb503e0-715b-47e8-adfd-ad4b7770f73b"},
		"grant_type": []string{"password"},
		"username":   []string{user},
		"password":   []string{password},
	}

	uri := fordAuth + "/v1.0/endpoint/default/token"
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout:       v.Helper.Client.Timeout,
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

func (v *Ford) request(uri string) (*http.Request, error) {
	if v.token == "" || time.Since(v.tokenValid) > 0 {
		if err := v.login(v.user, v.password); err != nil {
			return nil, err
		}
	}

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Content-type":   "application/json",
		"Application-Id": "71A3AD0A-CF46-4CCF-B473-FC7FE5BC4592",
		"Auth-Token":     v.token,
	})

	return req, err
}

// vehicles implements returns the list of user vehicles
func (v *Ford) vehicles() ([]string, error) {
	var br bmwVehiclesResponse
	uri := fmt.Sprintf("%s/me/vehicles/v2/", bmwAPI)

	var vehicles []string

	req, err := v.request(uri)
	if err == nil {
		err = v.DoJSON(req, &br)
	}

	if err == nil {
		for _, v := range br {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Ford) chargeState() (float64, error) {
	var resp struct {
		Vehiclestatus interface{}
	}

	uri := fmt.Sprintf("%s/api/vehicles/v4/%s/status", fordAPI, v.vin)

	req, err := v.request(uri)
	if err != nil {
		return 0, err
	}

	err = v.DoJSON(req, &resp)
	return 0, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Ford) ChargeState() (float64, error) {
	return v.chargeStateG()
}
