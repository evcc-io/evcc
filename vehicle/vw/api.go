package vw

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/andig/evcc/util/request"
)

// BaseURI is the VW api base URI
const BaseURI = "https://msg.volkswagen.de/fs-car"

// VehiclesResponse is the /usermanagement/users/v1/%s/%s/vehicles api
type VehiclesResponse struct {
	UserVehicles struct {
		Vehicle []string
	}
}

// ChargerResponse is the /bs/batterycharge/v1/%s/%s/vehicles/%s/charger api
type ChargerResponse struct {
	Charger struct {
		Status struct {
			BatteryStatusData struct {
				StateOfCharge struct {
					Content   int
					Timestamp string
				}
				RemainingChargingTime struct {
					Content   int
					Timestamp string
				}
			}
		}
	}
}

// API is the VW api client
type API struct {
	*request.Helper
	tokens         *Tokens
	authFlow       func() error
	refreshHeaders func() map[string]string
	brand, country string
	VIN            string
}

// NewAPI creates a new api client
func NewAPI(
	helper *request.Helper, tokens *Tokens,
	authFlow func() error, refreshHeaders func() map[string]string,
	vin, brand, country string,
) *API {
	v := &API{
		Helper:         helper,
		tokens:         tokens,
		authFlow:       authFlow,
		refreshHeaders: refreshHeaders,
		VIN:            vin,
		brand:          brand,
		country:        country,
	}
	return v
}

func (v *API) refreshToken() error {
	if v.tokens.RefreshToken == "" {
		return errors.New("missing refresh token")
	}

	data := url.Values(map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {v.tokens.RefreshToken},
		"scope":         {"sc2:fal"},
	})

	req, err := request.New(http.MethodPost, OauthTokenURI, strings.NewReader(data.Encode()), v.refreshHeaders())

	if err == nil {
		var tokens Tokens

		err = v.DoJSON(req, &tokens)
		if err == nil {
			v.tokens.AccessToken = tokens.AccessToken
			if tokens.RefreshToken != "" {
				v.tokens.RefreshToken = tokens.RefreshToken
			}
		}
	}

	return err
}

func (v *API) getJSON(uri string, res interface{}) error {
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.tokens.AccessToken,
	})

	if err == nil {
		err = v.DoJSON(req, &res)

		// token expired?
		if err != nil {
			// handle http 401
			if se, ok := err.(request.StatusError); ok && se.StatusCode() == http.StatusUnauthorized {
				// use refresh token
				if err = v.refreshToken(); err != nil {
					// re-run full auth flow
					err = v.authFlow()
				}
			}

			// retry original requests
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+v.tokens.AccessToken)
				err = v.DoJSON(req, &res)
			}
		}
	}

	return err
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() ([]string, error) {
	var res VehiclesResponse
	uri := fmt.Sprintf("%s/usermanagement/users/v1/%s/%s/vehicles", BaseURI, v.brand, v.country)
	err := v.getJSON(uri, &res)
	return res.UserVehicles.Vehicle, err
}

// Charger implements the /charger response
func (v *API) Charger() (ChargerResponse, error) {
	var res ChargerResponse
	uri := fmt.Sprintf("%s/bs/batterycharge/v1/%s/%s/vehicles/%s/charger", BaseURI, v.brand, v.country, v.VIN)
	err := v.getJSON(uri, &res)
	return res, err
}
