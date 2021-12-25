package porsche

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// API is an api.Vehicle implementation for Porsche PHEV cars
type API struct {
	log *util.Logger
	*request.Helper
	identity *Identity
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity *Identity) *API {
	impl := &API{
		log:      log,
		Helper:   request.NewHelper(log),
		identity: identity,
	}

	return impl
}

func (v *API) request(emobility bool, uri string) (*http.Request, error) {
	apiKey := OAuth2Config.ClientID
	token, err := v.identity.DefaultSource.Token()
	if emobility {
		apiKey = EmobilityOAuth2Config.ClientID
		token, err = v.identity.EmobilitySource.Token()
	}

	var req *http.Request
	if err == nil {
		req, err = request.New(http.MethodGet, uri, nil, map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", token.AccessToken),
			"apikey":        apiKey,
		})
	}

	return req, err
}

func (v *API) FindVehicle(vin string) (string, error) {
	token, err := v.identity.DefaultSource.Token()
	if err != nil {
		return "", err
	}

	vehiclesURL := "https://api.porsche.com/core/api/v3/de/de_DE/vehicles"
	req, err := request.New(http.MethodGet, vehiclesURL, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token.AccessToken),
		"apikey":        OAuth2Config.ClientID,
	})

	if err != nil {
		return "", err
	}

	var vehicles []VehicleResponse
	if err = v.DoJSON(req, &vehicles); err != nil {
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
	uri := fmt.Sprintf("%s/%s/pairing", vehiclesURL, foundVehicle.VIN)
	req, err = request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token.AccessToken),
		"apikey":        OAuth2Config.ClientID,
	})

	if err != nil {
		return "", err
	}

	var pairing VehiclePairingResponse
	if err = v.DoJSON(req, &pairing); err != nil {
		return "", err
	}

	if pairing.Status != "PAIRINGCOMPLETE" {
		return "", errors.New("vehicle is not paired with the My Porsche account")
	}

	// now check if we get any response at all for a status request
	// there are PHEV which do not provide any data, even thought they are PHEV
	uri = fmt.Sprintf("https://api.porsche.com/vehicle-data/de/de_DE/status/%s", foundVehicle.VIN)
	req, err = request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token.AccessToken),
		"apikey":        OAuth2Config.ClientID,
	})

	if err != nil {
		return "", err
	}

	if _, err = v.DoBody(req); err != nil {
		return "", errors.New("vehicle is not capable of providing data")
	}

	return foundVehicle.VIN, err
}

func (v *API) Capabilities(vin string) (CapabilitiesResponse, error) {
	// Note: As of 27.10.21 the capabilities API needs to be called AFTER a
	//   call to status() as it otherwise returns an HTTP 502 error.
	//   The reason is unknown, even when tested with 100% identical Headers.
	//   It seems to be a new backend related issue.

	if _, err := v.Status(vin); err != nil {
		return CapabilitiesResponse{}, err
	}

	uri := fmt.Sprintf("https://api.porsche.com/e-mobility/vcs/capabilities/%s", vin)

	req, err := v.request(true, uri)
	if err != nil {
		return CapabilitiesResponse{}, err
	}

	var res CapabilitiesResponse
	err = v.DoJSON(req, &res)
	return res, err
}

// Status implements the vehicle status response
func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("https://api.porsche.com/vehicle-data/de/de_DE/status/%s", vin)
	req, err := v.request(false, uri)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// EmobilityStatus implements the vehicle status response
func (v *API) EmobilityStatus(vin, model string) (EmobilityResponse, error) {
	var res EmobilityResponse

	uri := fmt.Sprintf("https://api.porsche.com/e-mobility/de/de_DE/%s/%s?timezone=Europe/Berlin", model, vin)
	req, err := v.request(true, uri)
	if err != nil {
		return res, err
	}

	err = v.DoJSON(req, &res)
	if err != nil && res.PcckErrorMessage != "" {
		err = errors.New(res.PcckErrorMessage)
	}

	return res, err
}
