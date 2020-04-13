package vehicle

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Credits to
// 	https://github.com/edent/Renault-Zoe-API/issues/18
// 	https://github.com/epenet/Renault-Zoe-API/blob/newapimockup/Test/MyRenault.py

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

type messageResponse struct {
	Message     string            `json:"message"`
	SessionInfo gigyaSessionInfo  `json:"sessionInfo"`
	IDToken     string            `json:"id_token"`
	Data        gigyaData         `json:"data"`
	Accounts    []kamereonAccount `json:"accounts"`
	AccessToken string            `json:"accessToken"`
}

type gigyaSessionInfo struct {
	CookieValue string `json:"cookieValue"`
}

type gigyaData struct {
	PersonID string `json:"personId"`
}

type kamereonAccount struct {
	AccountID string `json:"accountId"`
}

// Renault is an api.Vehicle implementation for Renault cars
type Renault struct {
	*embed
	*api.HTTPHelper
	user, password, vin                string
	gigya, kamereon                    configServer
	gigyaJwtToken, kamereonAccessToken string
	chargeStateG                       provider.FloatGetter
}

// NewRenaultFromConfig creates a new vehicle
func NewRenaultFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title                       string
		Capacity                    int64
		User, Password, Region, VIN string
		Cache                       time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

	if cc.Region == "" {
		cc.Region = "de_DE"
	}

	v := &Renault{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: api.NewHTTPHelper(api.NewLogger("zoe")),
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

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

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

			fmt.Println(personID)

			if err == nil {
				var accountID string
				accountID, err = v.kamereonPerson(personID)

				fmt.Println(accountID)

				if err == nil {
					var accessToken string
					accessToken, err = v.kamereonToken(accountID)

					fmt.Println(accessToken)

					v.kamereonAccessToken = accessToken
				}
			}
		}
	}

	return err
}

func (v *Renault) request(uri string, data url.Values, headers ...map[string]string) (messageResponse, error) {
	var tr messageResponse

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return tr, err
	}
	req.URL.RawQuery = data.Encode()

	for _, headers := range headers {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	_, err = v.RequestJSON(req, &tr)
	if err != nil {
		return tr, err
	}

	if tr.Message != "" {
		return tr, errors.New(tr.Message)
	}

	return tr, nil
}

func (v *Renault) sessionCookie(user, password string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.login", v.gigya.Target)

	data := url.Values{
		"loginID":  []string{user},
		"password": []string{password},
		"apiKey":   []string{v.gigya.APIKey},
	}

	tr, err := v.request(uri, data)
	return tr.SessionInfo.CookieValue, err
}

func (v *Renault) personID(sessionCookie string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.getAccountInfo", v.gigya.Target)

	data := url.Values{
		"oauth_token": []string{sessionCookie},
	}

	tr, err := v.request(uri, data)
	return tr.Data.PersonID, err
}

func (v *Renault) jwtToken(sessionCookie string) (string, error) {
	uri := fmt.Sprintf("%s/accounts.getJWT", v.gigya.Target)

	data := url.Values{
		"oauth_token": []string{sessionCookie},
		"fields":      []string{"data.personId,data.gigyaDataCenter"},
		"expiration":  []string{"900"},
	}

	tr, err := v.request(uri, data)
	return tr.IDToken, err
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
	uri := fmt.Sprintf("%s/commerce/v1/persons/%s", v.kamereon.Target, personID)

	data := url.Values{
		"country": []string{"DE"},
	}

	headers := v.kamereonHeaders()

	tr, err := v.request(uri, data, headers)
	if err != nil {
		return "", err
	}
	return tr.Accounts[0].AccountID, err
}

func (v *Renault) kamereonToken(accountID string) (string, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/token", v.kamereon.Target, accountID)

	data := url.Values{
		"country": []string{"DE"},
	}

	headers := v.kamereonHeaders()

	tr, err := v.request(uri, data, headers)
	return tr.AccessToken, err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Renault) chargeState() (float64, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/kmr/remote-services/car-adapter/v1/cars/%s/battery-status", v.kamereon.Target, v.vin)

	data := url.Values{
		"country": []string{"DE"},
	}

	headers := v.kamereonHeaders(map[string]string{"x-kamereon-authorization": "Bearer " + v.kamereonAccessToken})

	_, _ = v.request(uri, data, headers)

	return float64(0), nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Renault) ChargeState() (float64, error) {
	return v.chargeStateG()
}
