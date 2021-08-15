package skoda

import (
	"fmt"
	"net/http"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

const BaseURI = "https://api.connect.skoda-auto.cz/api"

// API is the Skoda api client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Client.Transport,
	}

	return v
}

func (v *API) getJSON(uri string, res interface{}) error {
	req, err := request.New(http.MethodGet, uri, nil, request.AcceptJSON)

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return err
}

// Vehicle is the /v2/garage/vehicles api
type Vehicle struct {
	ID, VIN       string
	LastUpdatedAt string
	Specification struct {
		Title, Brand, Model string
		Battery             struct {
			CapacityInKWh int
		}
	}
	// Connectivities
	// Capabilities
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() ([]string, error) {
	var res []Vehicle

	uri := fmt.Sprintf("%s/v2/garage/vehicles", BaseURI)
	err := v.getJSON(uri, &res)

	var vehicles []string
	if err == nil {
		for _, v := range res {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

// ChargerResponse is the /v1/charging/<vin>/status api
type ChargerResponse struct {
	Plug struct {
		ConnectionState string // Connected
		LockState       string // Unlocked
	}
	Charging struct {
		State                           string // Error
		RemainingToCompleteInSeconds    int64
		ChargingPowerInWatts            float64
		ChargingRateInKilometersPerHour float64
		ChargingType                    string // Invalid
		ChargeMode                      string // MANUAL
	}
	Battery struct {
		CruisingRangeElectricInMeters int64
		StateOfChargeInPercent        int
	}
}

// Charger implements the /v1/charging/<vin>/status response
func (v *API) Charger(vin string) (ChargerResponse, error) {
	var res ChargerResponse
	uri := fmt.Sprintf("%s/v1/charging/%s/status", BaseURI, vin)
	err := v.getJSON(uri, &res)
	return res, err
}
