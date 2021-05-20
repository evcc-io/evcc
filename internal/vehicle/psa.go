package vehicle

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/psa"
	"github.com/andig/evcc/util"
)

// https://github.com/flobz/psa_car_controller

func init() {
	registry.Add("citroen", NewCitroenFromConfig)
	registry.Add("opel", NewOpelFromConfig)
	registry.Add("peugeot", NewPeugeotFromConfig)
}

// NewCitroenFromConfig creates a new vehicle
func NewCitroenFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	log := util.NewLogger("citroen")
	return newPSA(log, "citroen.com", "clientsB2CCitroen", other)
}

// NewOpelFromConfig creates a new vehicle
func NewOpelFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	log := util.NewLogger("opel")
	return newPSA(log, "opel.com", "clientsB2COpel", other)
}

// NewPeugeotFromConfig creates a new vehicle
func NewPeugeotFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	log := util.NewLogger("peugeot")
	return newPSA(log, "peugeot.com", "clientsB2CPeugeot", other)
}

// PSA is an api.Vehicle implementation for PSA cars
type PSA struct {
	*embed
	*psa.Provider // provides the api implementations
}

// newPSA creates a new vehicle
func newPSA(log *util.Logger, brand, realm string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                  `mapstructure:",squash"`
		ClientID, ClientSecret string
		User, Password, VIN    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" || cc.ClientSecret == "" {
		return nil, errors.New("missing client id and secret (see https://github.com/flobz/psa_car_controller)")
	}

	v := &PSA{
		embed: &cc.embed,
	}

	api := psa.NewAPI(log, brand, realm, cc.ClientID, cc.ClientSecret)
	err := api.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	vehicles, err := api.Vehicles()
	if err != nil {
		return nil, err
	}

	var vid string
	if cc.VIN == "" && len(vehicles) == 1 {
		vid = vehicles[0].ID
	} else {
		for _, vehicle := range vehicles {
			if vehicle.VIN == strings.ToUpper(cc.VIN) {
				vid = vehicle.ID
			}
		}
	}

	if vid == "" {
		return nil, errors.New("vin not found")
	}

	v.Provider = psa.NewProvider(api, vid, cc.Cache)

	return v, err
}
