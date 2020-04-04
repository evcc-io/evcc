package vehicle

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

const (
	bmwURL = "https://b2vapi.bmwgroup.com/webapi"
)

type bmwTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type bmwStatusResponse struct {
	VehicleStatus bmwVehicleStatus `json:"vehicleStatus"`
}

type bmwVehicleStatus struct {
	ConnectionStatus      string `json:"connectionStatus"`
	ChargingStatus        string `json:"chargingStatus"`
	ChargingLevelHv       int    `json:"chargingLevelHv"`
	ChargingTimeRemaining int    `json:"chargingTimeRemaining"`
}

// BMW is an api.Vehicle implementation for BMW cars
type BMW struct {
	*embed
	user, password, vin string
	token, refreshToken string
	tokenValid          time.Time
	chargeStateG        provider.FloatGetter
}

// NewBMWFromConfig creates a new vehicle
func NewBMWFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

	v := &BMW{
		embed:    &embed{cc.Title, cc.Capacity},
		user:     cc.User,
		password: cc.Password,
		vin:      cc.VIN,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v
}

func (v *BMW) apiURL(service string) string {
	return fmt.Sprintf("%s/%s", bmwURL, service)
}

func (v *BMW) authHeader() string {
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", v.user, v.password)))
	return fmt.Sprintf("Basic %s", token)
}

func (v *BMW) login(user, password string) error {
	uri := v.apiURL("oauth/token")

	data := url.Values{
		"grant_type": []string{"password"},
		"username":   []string{user},
		"password":   []string{password},
		"scope":      []string{"remote_services vehicle_data"},
	}

	req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", v.authHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response %d: %s", resp.StatusCode, string(b))
	}

	var tr bmwTokenResponse
	if err = json.Unmarshal(b, &tr); err != nil {
		return err
	}

	v.token = tr.AccessToken
	v.refreshToken = tr.RefreshToken
	v.tokenValid = time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second - tokenValidMargin)

	return nil
}

// @TODO implement refresh_token
func (v *BMW) request(uri string) (*http.Request, error) {
	// token invalid or expired
	if v.token == "" || time.Now().After(v.tokenValid) {
		if err := v.login(v.user, v.password); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return req, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v.token))

	return req, nil
}

// chargeState implements the Vehicle.ChargeState interface
func (v *BMW) chargeState() (float64, error) {
	uri := v.apiURL(fmt.Sprintf("v1/user/vehicles/%s/status", v.vin))
	req, err := v.request(uri)
	if err != nil {
		return 0, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var br bmwStatusResponse
	err = json.Unmarshal(b, &br)

	return float64(br.VehicleStatus.ChargingLevelHv), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *BMW) ChargeState() (float64, error) {
	return v.chargeStateG()
}
