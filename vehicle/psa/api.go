package psa

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://developer.groupe-psa.io/webapi/b2c/api-reference/specification

// BaseURL is the API base url
const BaseURL = "https://api.groupe-psa.com/connectedcar/v4"

// API is an api.Vehicle implementation for PSA cars
type API struct {
	*request.Helper
	realm string
	id    string
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity oauth2.TokenSource, realm, id string) *API {
	v := &API{
		Helper: request.NewHelper(log),
		realm:  realm,
		id:     id,
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Client.Transport,
	}

	return v
}

// Vehicle is a single vehicle
type Vehicle struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Pictures []string `json:"pictures"`
	VIN      string   `json:"vin"`
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() ([]Vehicle, error) {
	data := url.Values{
		"client_id": []string{v.id},
	}

	uri := fmt.Sprintf("%s/user/vehicles?%s", BaseURL, data.Encode())
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":             "application/hal+json",
		"X-Introspect-Realm": v.realm,
	})

	var res struct {
		Embedded struct {
			Vehicles []Vehicle
		} `json:"_embedded"`
	}
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res.Embedded.Vehicles, err
}

// Status is the /status response
type Status struct {
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
	Preconditionning struct {
		AirConditioning struct {
			UpdatedAt time.Time
			Status    string // Disabled
		}
	}
	Energy []Energy
}

// Energy is the /status partial energy response
type Energy struct {
	UpdatedAt time.Time
	Type      string // Fuel/Electric
	Level     int
	Autonomy  int
	Charging  struct {
		Plugged         bool
		Status          string // InProgress
		RemainingTime   Duration
		ChargingRate    int
		ChargingMode    string // "Slow"
		NextDelayedTime Duration
	}
}

// Status implements the /vehicles/<vid>/status response
func (v *API) Status(vid string) (Status, error) {
	data := url.Values{
		"client_id": []string{v.id},
	}

	// BaseURL is the API base url
	uri := fmt.Sprintf("%s/user/vehicles/%s/status?%s", BaseURL, vid, data.Encode())
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":             "application/hal+json",
		"X-Introspect-Realm": v.realm,
	})

	var status Status
	if err == nil {
		err = v.DoJSON(req, &status)
	}

	return status, err
}
