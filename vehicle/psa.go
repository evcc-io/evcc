package vehicle

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/psa"
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
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		User, Password, VIN    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &PSA{
		embed: &embed{cc.Title, cc.Capacity},
	}

	api := psa.NewAPI(log, brand, realm, cc.ClientID, cc.ClientSecret)

	err := api.Login(cc.User, cc.Password)
	if err == nil {
		if cc.VIN == "" {
			cc.VIN, err = findVehicle(api.Vehicles())
			if err == nil {
				log.DEBUG.Printf("found vehicle: %v", cc.VIN)
			}
		}
	}

	v.Provider = psa.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)

	return v, err
}
