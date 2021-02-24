package psa

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://github.com/flobz/psa_car_controller
// https://github.com/snaptec/openWB/blob/master/modules/soc_psa/psasoc.py

// BaseURL is the API base url
const BaseURL = "https://api.groupe-psa.com/connectedcar"

// type oauth2Token struct {
// 	oauth2.Token
// 	IDToken string `json:"id_token"`
// }

// API is an api.Vehicle implementation for PSA cars
type API struct {
	*request.Helper
	brand, realm     string
	clientID, secret string
	token            oauth2.Token
	// token oauth2Token
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, brand, realm, id, secret string) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		brand:    brand,
		realm:    realm,
		clientID: id,
		secret:   secret,
	}
	return v
}

// Login performs the login
func (v *API) Login(user, password string) error {
	data := url.Values{
		"realm":      []string{v.realm},
		"scope":      []string{"openid profile"},
		"grant_type": []string{"password"},
		"username":   []string{user},
		"password":   []string{password},
	}

	auth := fmt.Sprintf("%s:%s", v.clientID, v.secret)

	uri := fmt.Sprintf("https://idpcvs.%s/am/oauth2/access_token", v.brand)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(auth)),
	})

	if err == nil {
		err = v.DoJSON(req, &v.token)
	}

	fmt.Printf("\n%+v\n", v.token)

	return err
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() (res []string, err error) {
	data := url.Values{
		"client_id": []string{v.clientID},
	}

	// BaseURL is the API base url
	uri := fmt.Sprintf("%s/v4/user/vehicles?%s", BaseURL, data.Encode())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":             "application/hal+json",
		"Authorization":      "Bearer " + v.token.AccessToken,
		"X-Introspect-Realm": v.realm,
	})

	var vehicles map[string]interface{}
	if err == nil {
		err = v.DoJSON(req, &vehicles)

		// for _, v := range vehicles.Data {
		// 	res = append(res, v.VIN)
		// }
	}

	return res, err
}

// Status is the /vehicles/<vin>/status response
type Status struct{}

// Status implements the /vehicles/<vin>/status response
func (v *API) Status(vin string) (Status, error) {
	data := url.Values{
		"client_id": []string{v.clientID},
	}

	// BaseURL is the API base url
	uri := fmt.Sprintf("%s/v4/user/vehicles/%s/status?%s", BaseURL, vin, data.Encode())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":             "application/hal+json",
		"Authorization":      "Bearer " + v.token.AccessToken,
		"X-Introspect-Realm": v.realm,
	})

	var vehicles map[string]interface{}
	if err == nil {
		err = v.DoJSON(req, &vehicles)

		// for _, v := range vehicles.Data {
		// 	res = append(res, v.VIN)
		// }
	}

	return Status{}, err
}
