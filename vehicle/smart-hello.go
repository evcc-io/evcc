package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/smart/hello"
)

// SmartHello is an api.Vehicle implementation for Smart Hello cars
type SmartHello struct {
	*embed
	*hello.Provider
}

func init() {
	registry.Add("smart-hello", NewSmartHelloFromConfig)
}

// NewSmartHelloFromConfig creates a new vehicle
func NewSmartHelloFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Cache          time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("smart-hello").Redact(cc.User, cc.Password, cc.VIN)

	v := &SmartHello{
		embed: &cc.embed,
	}

	identity, err := hello.NewIdentity(log, cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := hello.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		v.Provider = hello.NewProvider(log, api, cc.VIN, cc.Cache)
	}

	return v, err
}
