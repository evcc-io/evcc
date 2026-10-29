package livewire

import (
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

var BaseURL = "https://mobileapi.livewire.com/api"

const Brand = "LiveWire"

// API is the LiveWire mobile api client
type API struct {
	*request.Helper
	deviceUUID string
}

// NewAPI creates a new api client authenticated by identity
func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:     request.NewHelper(log),
		deviceUUID: identity.deviceUUID,
	}

	v.Client.Transport = identity.Transport(v.Client.Transport)

	return v
}

// Vehicles returns the bikes of the account including their pairing status with this device
func (v *API) Vehicles() ([]Bike, error) {
	var res BikesResponse
	uri := fmt.Sprintf("%s/getAllbikes/pairingStatus?deviceUUID=%s", BaseURL, url.QueryEscape(v.deviceUUID))
	err := v.getJSON(uri, &res)
	return res.Bikes, err
}

// Status returns the charging status of the bike
func (v *API) Status(bikeID string) (ChargingStatus, error) {
	var res ChargingStatusResponse
	err := v.getJSON(fmt.Sprintf("%s/bikes/%s/charging/status", BaseURL, bikeID), &res)
	return res.BikeChargingData, err
}

// Position returns the bike location
func (v *API) Position(bikeID string) (Position, error) {
	var res LocationResponse
	err := v.getJSON(fmt.Sprintf("%s/bike/%s/location", BaseURL, bikeID), &res)
	return res.Data, err
}

type envelope interface {
	Err() error
}

// getJSON executes a GET and surfaces the error envelope, which arrives with HTTP 200
func (v *API) getJSON(uri string, res envelope) error {
	err := v.GetJSON(uri, res)

	if envErr := res.Err(); envErr != nil {
		return envErr
	}

	return err
}
