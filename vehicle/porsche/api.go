package porsche

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const (
	ApiURI = "https://api.porsche.com"
)

// API is an api.Vehicle implementation for Porsche PHEV cars
type API struct {
	log *util.Logger
	*request.Helper
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	v := &API{
		log:    log,
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"apikey": OAuth2Config.ClientID,
		}),
	}

	return v
}

func (v *API) FindVehicle(vin string) (string, error) {
	var vehicles []VehicleResponse

	uri := fmt.Sprintf("%s/core/api/v3/de/de_DE/vehicles", ApiURI)
	if err := v.GetJSON(uri, &vehicles); err != nil {
		return "", err
	}

	var foundVehicle VehicleResponse

	if vin == "" && len(vehicles) == 1 {
		foundVehicle = vehicles[0]
	} else {
		for _, vehicleItem := range vehicles {
			if vehicleItem.VIN == strings.ToUpper(vin) {
				foundVehicle = vehicleItem
			}
		}
	}
	if foundVehicle.VIN == "" {
		return "", errors.New("vin not found")
	}

	v.log.DEBUG.Printf("found vehicle: %v", foundVehicle.VIN)

	// check if vehicle is paired
	var pairing VehiclePairingResponse
	uri = fmt.Sprintf("%s/%s/pairing", uri, foundVehicle.VIN)
	if err := v.GetJSON(uri, &pairing); err != nil {
		return "", err
	}

	if pairing.Status != "PAIRINGCOMPLETE" {
		return "", errors.New("vehicle is not paired with the My Porsche account")
	}

	// now check if we get any response at all for a status request
	// there are PHEV which do not provide any data, even thought they are PHEV
	uri = fmt.Sprintf("%s/vehicle-data/de/de_DE/status/%s", ApiURI, foundVehicle.VIN)
	if err := v.GetJSON(uri, nil); err != nil {
		return "", errors.New("vehicle is not capable of providing data")
	}

	return foundVehicle.VIN, nil
}

// Status implements the vehicle status response
func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse
	uri := fmt.Sprintf("%s/vehicle-data/de/de_DE/status/%s", ApiURI, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}
