package psa

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/thoas/go-funk"
	"golang.org/x/oauth2"
)

// https://github.com/flobz/psa_car_controller
// https://developer.groupe-psa.io/webapi/b2c/api-reference/specification

// BaseURL is the API base url
const BaseURL = "https://api.groupe-psa.com/connectedcar"

// API is an api.Vehicle implementation for PSA cars
type API struct {
	*request.Helper
	brand, realm string
	id, secret   string // client
	token        oauth2.Token
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, brand, realm, id, secret string) *API {
	v := &API{
		Helper: request.NewHelper(log),
		brand:  brand,
		realm:  realm,
		id:     id,
		secret: secret,
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

	auth := fmt.Sprintf("%s:%s", v.id, v.secret)

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

// Vehicle is a single vehicle
type Vehicle struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Pictures []string `json:"pictures"`
	VIN      string   `json:"vin"`
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() ([]string, error) {
	data := url.Values{
		"client_id": []string{v.id},
	}

	// BaseURL is the API base url
	uri := fmt.Sprintf("%s/v4/user/vehicles?%s", BaseURL, data.Encode())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":             "application/hal+json",
		"Authorization":      "Bearer " + v.token.AccessToken,
		"X-Introspect-Realm": v.realm,
	})

	var res []string
	if err == nil {
		var vehicles struct {
			Embedded struct {
				Vehicles []Vehicle
			} `json:"_embedded"`
		}

		if err = v.DoJSON(req, &vehicles); err == nil {
			res = funk.Map(vehicles.Embedded.Vehicles, func(v Vehicle) string {
				return v.VIN
			}).([]string)
		}
	}

	return res, err
}

// Energy struct
type Energy struct {
	Battery struct {
		Capacity int64
		Health   struct {
			Capacity   int64
			Resistance int64
		}
	}
	Charging struct {
		ChargingMode    string
		ChargingRate    int64
		NextDelayedTime string
		Plugged         bool
		RemainingTime   string
		Status          string
	}
}

// Status implements the /vehicles/<vin>/status response
func (v *API) Status(vin string) (Energy, error) {
	data := url.Values{
		"client_id": []string{v.id},
	}

	// BaseURL is the API base url
	uri := fmt.Sprintf("%s/v4/user/vehicles/%s/status?%s", BaseURL, vin, data.Encode())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":             "application/hal+json",
		"Authorization":      "Bearer " + v.token.AccessToken,
		"X-Introspect-Realm": v.realm,
	})

	var status struct {
		Energy [][]Energy
	}

	if err == nil {
		if err = v.DoJSON(req, &status); err == nil {
			for _, e := range status.Energy {
				for _, energy := range e {
					return energy, nil
				}
			}
		}
	}

	return Energy{}, err
}
