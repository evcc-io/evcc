package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/psa"
)

// https://github.com/TA2k/ioBroker.psa

func init() {
	registry.Add("citroen", NewCitroenFromConfig)
	registry.Add("ds", NewDSFromConfig)
	registry.Add("opel", NewOpelFromConfig)
	registry.Add("peugeot", NewPeugeotFromConfig)
}

// NewCitroenFromConfig creates a new vehicle
func NewCitroenFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	log := util.NewLogger("citroen")
	return newPSA(log,
		"citroen.com", "clientsB2CCitroen",
		"5364defc-80e6-447b-bec6-4af8d1542cae", "iE0cD8bB0yJ0dS6rO3nN1hI2wU7uA5xR4gP7lD6vM0oH0nS8dN",
		other)
}

// NewDSFromConfig creates a new vehicle
func NewDSFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	log := util.NewLogger("ds")
	return newPSA(log,
		"driveds.com", "clientsB2CDS",
		"cbf74ee7-a303-4c3d-aba3-29f5994e2dfa", "X6bE6yQ3tH1cG5oA6aW4fS6hK0cR0aK5yN2wE4hP8vL8oW5gU3",
		other)
}

// NewOpelFromConfig creates a new vehicle
func NewOpelFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	log := util.NewLogger("opel")
	return newPSA(log,
		"opel.com", "clientsB2COpel",
		"07364655-93cb-4194-8158-6b035ac2c24c", "F2kK7lC5kF5qN7tM0wT8kE3cW1dP0wC5pI6vC0sQ5iP5cN8cJ8",
		other)
}

// NewPeugeotFromConfig creates a new vehicle
func NewPeugeotFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	log := util.NewLogger("peugeot")
	return newPSA(log,
		"peugeot.com", "clientsB2CPeugeot",
		"1eebc2d5-5df3-459b-a624-20abfcf82530", "T5tP7iS0cO8sC0lA2iE2aR7gK6uE5rF3lJ8pC3nO1pR7tL8vU1",
		other)
}

// PSA is an api.Vehicle implementation for PSA cars
type PSA struct {
	*embed
	*psa.Provider // provides the api implementations
}

// newPSA creates a new vehicle
func newPSA(log *util.Logger, brand, realm, id, secret string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		Credentials         ClientCredentials
		User, Password, VIN string
		Cache               time.Duration
	}{
		Credentials: ClientCredentials{
			ID:     id,
			Secret: secret,
		},
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &PSA{
		embed: &cc.embed,
	}

	log.Redact(cc.User, cc.Password, cc.VIN)
	identity := psa.NewIdentity(log, brand, cc.Credentials.ID, cc.Credentials.Secret)

	if err := identity.Login(cc.User, cc.Password); err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := psa.NewAPI(log, identity, realm, cc.Credentials.ID)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v psa.Vehicle) string {
			return v.VIN
		},
	)

	if err != nil {
		return nil, err
	}

	v.Provider = psa.NewProvider(api, vehicle.ID, cc.Cache)

	return v, err
}
