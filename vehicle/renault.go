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
	SessionInfo gigyaSessionInfo `json:"sessionInfo"` // /accounts.login
	IDToken     string           `json:"id_token"`    // /accounts.getJWT
	Data        gigyaData        `json:"data"`        // /accounts.getAccountInfo
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
	Attributes batteryAttributes `json:"attributes"`
}

type batteryAttributes struct {
	ChargeStatus       int    `json:"chargeStatus"`
	InstantaneousPower int    `json:"instantaneousPower"`
	RangeHvacOff       int    `json:"rangeHvacOff"`
	BatteryLevel       int    `json:"batteryLevel"`
	BatteryTemperature int    `json:"batteryTemperature"`
	PlugStatus         int    `json:"plugStatus"`
	LastUpdateTime     string `json:"lastUpdateTime"`
	ChargePower        int    `json:"chargePower"`
}

// Renault is an api.Vehicle implementation for Renault cars
type Renault struct {
	*embed
	*util.HTTPHelper
	user, password, vin                string
	gigya, kamereon                    configServer
	gigyaJwtToken, kamereonAccessToken string
	accountID                          string
	chargeStateG                       provider.FloatGetter
}

// NewRenaultFromConfig creates a new vehicle
func NewRenaultFromConfig(log *util.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title                       string
		Capacity                    int64
		User, Password, Region, VIN string
		Cache                       time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	if cc.Region == "" {
		cc.Region = "de_DE"
	}

	logger := util.NewLogger("zoe")

	v := &Renault{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(logger),
		user:       cc.User,
		password:   cc.Password,
		vin:        cc.VIN,
	}

	err := v.apiKeys(cc.Region)
	if err == nil {
		err = v.authFlow()
	}
	if err != nil {
		v.HTTPHelper.Log.FATAL.Fatalf("cannot create renault: %v", err)
	}

	v.chargeStateG = provider.NewCached(logger, v.chargeState, cc.Cache).FloatGetter()

	return v
}

func (v *Renault) apiKeys(region string) error {
	uri := fmt.Sprintf(keyStore, region)

	var cr configResponse
	_, err := v.GetJSON(uri, &cr)

	v.gigya = cr.Servers.GigyaProd
	v.kamereon = cr.Servers.WiredProd

	return err
}

func (v *Renault) authFlow() error {
	sessionCookie, err := v.sessionCookie(v.user, v.password)

	if err == nil {
		v.gigyaJwtToken, err = v.jwtToken(sessionCookie)

		if err == nil {
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

				// find VIN
				if v.vin == "" {
					v.vin, err = v.kamereonVehicles(v.accountID)
					if v.vin == "" {
						return errors.New("missing vin")
					}
				}

				if err == nil {
					v.kamereonAccessToken, err = v.kamereonToken(v.accountID)
					if v.kamereonAccessToken == "" {
						return errors.New("missing kamereon access token")
					}
				}
			}
		}
	}

	return err
}

func (v *Renault) request(uri string, data url.Values, headers ...map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return req, err
	}
	req.URL.RawQuery = data.Encode()

	for _, headers := range headers {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	return req, nil
}

func (v *Renault) sessionCookie(user, password string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.login", v.gigya.Target)

	data := url.Values{
		"loginID":  []string{user},
		"password": []string{password},
		"apiKey":   []string{v.gigya.APIKey},
	}

	req, err := v.request(uri, data)

	var gr gigyaResponse
	if err == nil {
		_, err = v.RequestJSON(req, &gr)
	}

	return gr.SessionInfo.CookieValue, err
}

func (v *Renault) personID(sessionCookie string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.getAccountInfo", v.gigya.Target)

	data := url.Values{
		"oauth_token": []string{sessionCookie},
	}

	req, err := v.request(uri, data)

	var gr gigyaResponse
	if err == nil {
		_, err = v.RequestJSON(req, &gr)
	}

	return gr.Data.PersonID, err
}

func (v *Renault) jwtToken(sessionCookie string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.getJWT", v.gigya.Target)

	data := url.Values{
		"oauth_token": []string{sessionCookie},
		"fields":      []string{"data.personId,data.gigyaDataCenter"},
		"expiration":  []string{"900"},
	}

	req, err := v.request(uri, data)

	var gr gigyaResponse
	if err == nil {
		_, err = v.RequestJSON(req, &gr)
	}

	return gr.IDToken, err
}

func (v *Renault) kamereonHeaders(additional ...map[string]string) map[string]string {
	headers := map[string]string{
		"x-gigya-id_token": v.gigyaJwtToken,
		"apikey":           v.kamereon.APIKey,
	}

	for _, h := range additional {
		for k, v := range h {
			headers[k] = v
		}
	}

	return headers
}

func (v *Renault) kamereonPerson(personID string) (string, error) {
	var kr kamereonResponse
	uri := fmt.Sprintf("%s/commerce/v1/persons/%s", v.kamereon.Target, personID)

	data := url.Values{"country": []string{"DE"}}
	headers := v.kamereonHeaders()

	req, err := v.request(uri, data, headers)
	if err == nil {
		_, err = v.RequestJSON(req, &kr)

		if len(kr.Accounts) == 0 {
			return "", nil
		}
	}

	return kr.Accounts[0].AccountID, err
}

func (v *Renault) kamereonVehicles(accountID string) (string, error) {
	var kr kamereonResponse
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/vehicles", v.kamereon.Target, accountID)

	data := url.Values{"country": []string{"DE"}}
	headers := v.kamereonHeaders()

	req, err := v.request(uri, data, headers)
	if err == nil {
		_, err = v.RequestJSON(req, &kr)

		for _, v := range kr.VehicleLinks {
			if strings.ToUpper(v.Status) == "ACTIVE" {
				return v.VIN, nil
			}
		}
	}

	return "", err
}

func (v *Renault) kamereonToken(accountID string) (string, error) {
	var kr kamereonResponse
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/token", v.kamereon.Target, accountID)

	data := url.Values{"country": []string{"DE"}}
	headers := v.kamereonHeaders()

	req, err := v.request(uri, data, headers)
	if err == nil {
		_, err = v.RequestJSON(req, &kr)
	}

	return kr.AccessToken, err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Renault) chargeState() (float64, error) {
	var kr kamereonResponse
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/battery-status", v.kamereon.Target, v.accountID, v.vin)

	data := url.Values{"country": []string{"DE"}}
	headers := v.kamereonHeaders(map[string]string{"x-kamereon-authorization": "Bearer " + v.kamereonAccessToken})

	req, err := v.request(uri, data, headers)
	if err == nil {
		_, err = v.RequestJSON(req, &kr)
	}
	return float64(kr.Data.Attributes.BatteryLevel), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Renault) ChargeState() (float64, error) {
	return v.chargeStateG()
}
