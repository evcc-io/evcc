package vehicle

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// Credits to
// 	https://github.com/edent/Renault-Zoe-API/issues/18
// 	https://github.com/epenet/Renault-Zoe-API/blob/newapimockup/Test/MyRenault.py
//  https://github.com/jamesremuscat/pyze

const (
	keyStore = "https://renault-wrd-prod-1-euw1-myrapp-one.s3-eu-west-1.amazonaws.com/configuration/android/config_%s.json"
)

type configResponse struct {
	Servers configServers
}

type configServers struct {
	GigyaProd configServer `json:"gigyaProd"`
	WiredProd configServer `json:"wiredProd"`
}

type configServer struct {
	Target string `json:"target"`
	APIKey string `json:"apikey"`
}

type gigyaResponse struct {
	ErrorCode    int              `json:"errorCode"`    // /accounts.login
	ErrorMessage string           `json:"errorMessage"` // /accounts.login
	SessionInfo  gigyaSessionInfo `json:"sessionInfo"`  // /accounts.login
	IDToken      string           `json:"id_token"`     // /accounts.getJWT
	Data         gigyaData        `json:"data"`         // /accounts.getAccountInfo
}

type gigyaSessionInfo struct {
	CookieValue string `json:"cookieValue"`
}

type gigyaData struct {
	PersonID string `json:"personId"`
}

type kamereonResponse struct {
	Accounts     []kamereonAccount `json:"accounts"`     // /commerce/v1/persons/%s
	AccessToken  string            `json:"accessToken"`  // /commerce/v1/accounts/%s/kamereon/token
	VehicleLinks []kamereonVehicle `json:"vehicleLinks"` // /commerce/v1/accounts/%s/vehicles
	Data         kamereonData      `json:"data"`         // /commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/battery-status
}

type kamereonAccount struct {
	AccountID string `json:"accountId"`
}

type kamereonVehicle struct {
	Brand  string `json:"brand"`
	VIN    string `json:"vin"`
	Status string `json:"status"`
}

type kamereonData struct {
	Attributes attributes `json:"attributes"`
}

type attributes struct {
	Timestamp          string  `json:"timestamp"`
	ChargingStatus     float32 `json:"chargingStatus"`
	InstantaneousPower int     `json:"instantaneousPower"`
	RangeHvacOff       int     `json:"rangeHvacOff"`
	BatteryAutonomy    int     `json:"batteryAutonomy"`
	BatteryLevel       int     `json:"batteryLevel"`
	BatteryTemperature int     `json:"batteryTemperature"`
	PlugStatus         int     `json:"plugStatus"`
	LastUpdateTime     string  `json:"lastUpdateTime"`
	ChargePower        int     `json:"chargePower"`
	RemainingTime      *int    `json:"chargingRemainingTime"`
}

// Renault is an api.Vehicle implementation for Renault cars
type Renault struct {
	*embed
	*request.Helper
	user, password, vin string
	gigya, kamereon     configServer
	gigyaJwtToken       string
	accountID           string
	apiG                func() (interface{}, error)
}

func init() {
	registry.Add("renault", NewRenaultFromConfig)
}

// NewRenaultFromConfig creates a new vehicle
func NewRenaultFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                       string
		Capacity                    int64
		User, Password, Region, VIN string
		Cache                       time.Duration
	}{
		Region: "de_DE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("renault")

	v := &Renault{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	err := v.apiKeys(cc.Region)
	if err == nil {
		err = v.authFlow()
	}

	if err == nil && cc.VIN == "" {
		v.vin, err = findVehicle(v.kamereonVehicles(v.accountID))
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	v.apiG = provider.NewCached(v.batteryAPI, cc.Cache).InterfaceGetter()

	return v, err
}

func (v *Renault) apiKeys(region string) error {
	uri := fmt.Sprintf(keyStore, region)

	var cr configResponse
	err := v.GetJSON(uri, &cr)

	v.gigya = cr.Servers.GigyaProd
	v.kamereon = cr.Servers.WiredProd

	return err
}

func (v *Renault) authFlow() error {
	sessionCookie, err := v.sessionCookie(v.user, v.password)

	if err == nil {
		v.gigyaJwtToken, err = v.jwtToken(sessionCookie)

		if err == nil {
			if v.accountID != "" {
				// personID, accountID and VIN have already been read, skip remainder of flow
				return nil
			}

			var personID string
			personID, err = v.personID(sessionCookie)
			if personID == "" {
				return errors.New("missing personID")
			}

			if err == nil {
				v.accountID, err = v.kamereonPerson(personID)
				if v.accountID == "" {
					return errors.New("missing accountID")
				}
			}
		}
	}

	return err
}

func (v *Renault) request(uri string, data url.Values, headers ...map[string]string) (*http.Request, error) {
	req, err := request.New(http.MethodGet, uri, nil, headers...)
	if err == nil {
		req.URL.RawQuery = data.Encode()
	}

	return req, err
}

func (v *Renault) sessionCookie(user, password string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.login", v.gigya.Target)

	data := url.Values{
		"loginID":  []string{user},
		"password": []string{password},
		"apiKey":   []string{v.gigya.APIKey},
	}

	req, err := v.request(uri, data)

	var res gigyaResponse
	if err == nil {
		err = v.DoJSON(req, &res)
		if err == nil && res.ErrorCode > 0 {
			err = errors.New(res.ErrorMessage)
		}
	}

	return res.SessionInfo.CookieValue, err
}

func (v *Renault) personID(sessionCookie string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.getAccountInfo", v.gigya.Target)

	data := url.Values{
		"apiKey":      []string{v.gigya.APIKey},
		"login_token": []string{sessionCookie},
	}

	req, err := v.request(uri, data)

	var res gigyaResponse
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res.Data.PersonID, err
}

func (v *Renault) jwtToken(sessionCookie string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.getJWT", v.gigya.Target)

	data := url.Values{
		"apiKey":      []string{v.gigya.APIKey},
		"login_token": []string{sessionCookie},
		"fields":      []string{"data.personId,data.gigyaDataCenter"},
		"expiration":  []string{"900"},
	}

	req, err := v.request(uri, data)

	var res gigyaResponse
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res.IDToken, err
}

func (v *Renault) kamereonRequest(uri string) (kamereonResponse, error) {
	data := url.Values{"country": []string{"DE"}}
	headers := map[string]string{
		"x-gigya-id_token": v.gigyaJwtToken,
		"apikey":           v.kamereon.APIKey,
	}

	var res kamereonResponse
	req, err := v.request(uri, data, headers)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (v *Renault) kamereonPerson(personID string) (string, error) {
	uri := fmt.Sprintf("%s/commerce/v1/persons/%s", v.kamereon.Target, personID)
	res, err := v.kamereonRequest(uri)

	if len(res.Accounts) == 0 {
		return "", err
	}

	return res.Accounts[0].AccountID, err
}

func (v *Renault) kamereonVehicles(accountID string) ([]string, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/vehicles", v.kamereon.Target, accountID)
	res, err := v.kamereonRequest(uri)

	var vehicles []string
	if err == nil {
		for _, v := range res.VehicleLinks {
			if strings.ToUpper(v.Status) == "ACTIVE" {
				vehicles = append(vehicles, v.VIN)
			}
		}
	}

	return vehicles, err
}

// batteryAPI provides battery api response
func (v *Renault) batteryAPI() (interface{}, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v2/cars/%s/battery-status", v.kamereon.Target, v.accountID, v.vin)
	res, err := v.kamereonRequest(uri)

	// repeat auth if error
	if err != nil {
		if err = v.authFlow(); err == nil {
			res, err = v.kamereonRequest(uri)
		}
	}

	return res, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Renault) ChargeState() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(kamereonResponse); err == nil && ok {
		return float64(res.Data.Attributes.BatteryLevel), nil
	}

	return 0, err
}

// Status implements the Vehicle.Status interface
func (v *Renault) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.apiG()
	if res, ok := res.(kamereonResponse); err == nil && ok {
		if res.Data.Attributes.PlugStatus > 0 {
			status = api.StatusB
		}
		if res.Data.Attributes.ChargingStatus > 1.0 {
			status = api.StatusC
		}
	}

	return status, err
}

// Range implements the Vehicle.Range interface
func (v *Renault) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(kamereonResponse); err == nil && ok {
		return int64(res.Data.Attributes.BatteryAutonomy), nil
	}

	return 0, err
}

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *Renault) FinishTime() (time.Time, error) {
	res, err := v.apiG()

	if res, ok := res.(kamereonResponse); err == nil && ok {
		var timestamp time.Time
		if err == nil {
			timestamp, err = time.Parse(time.RFC3339, res.Data.Attributes.Timestamp)
		}

		if res.Data.Attributes.RemainingTime == nil {
			return time.Time{}, api.ErrNotAvailable
		}

		return timestamp.Add(time.Duration(*res.Data.Attributes.RemainingTime) * time.Minute), err
	}

	return time.Time{}, err
}
