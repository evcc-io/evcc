package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/zero"
)

// MG is an api.Vehicle implementation for probably all SAIC cars
type ZeroMotorcycle struct {
	*embed
	*zero.Provider // provides the api implementations
}

func init() {
	registry.Add("zero", NewZeroFromConfig)
}

// NewBMWFromConfig creates a new vehicle
func NewZeroFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("Zero").Redact(cc.User, cc.Password)
	identity := zero.NewIdentity(log, cc.User, cc.Password)

	if err := identity.Login(); err != nil {
		return nil, err
	}

	api := zero.NewAPI(log, identity)

	v := &ZeroMotorcycle{
		embed:    &cc.embed,
		Provider: zero.NewProvider(api, cc.Cache),
	}

	return v, nil
}
