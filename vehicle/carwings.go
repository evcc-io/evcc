package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/carwings"
)

// CarWings is an api.Vehicle implementation for CarWings cars
type CarWings struct {
	*embed
	*carwings.Provider
}

func init() {
	registry.Add("carwings", NewCarWingsFromConfig)
}

// NewCarWingsFromConfig creates a new vehicle
func NewCarWingsFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                       `mapstructure:",squash"`
		User, Password, Region, VIN string
		Cache                       time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("carwings").Redact(cc.User, cc.Password, cc.VIN)

	v := &CarWings{
		embed: &cc.embed,
	}

	identity := carwings.NewIdentity(log)
	login, err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := carwings.NewAPI(log, login.VehicleInfo.CustomSessionID, login.CustomerInfo.Timezone)
	err = CheckVIN(login, cc.VIN)
	if err != nil {
		return nil, err
	}

	v.Provider = carwings.NewProvider(api, cc.VIN, cc.Cache)

	return v, nil
}

func CheckVIN(login carwings.LoginResponse, VIN string) error {
	var isPresent bool
	switch {
	case len(login.VehicleInfos) > 0:
		for _, vi := range login.VehicleInfos {
			if VIN == vi.VIN {
				isPresent = true
			}
		}

	case len(login.VehicleInfoList.VehicleInfos) > 0:
		for _, vi := range login.VehicleInfoList.VehicleInfos {
			if VIN == vi.VIN {
				isPresent = true
			}
		}

	case len(login.CustomerInfo.VehicleInfo.VIN) > 0:
		if VIN == login.CustomerInfo.VehicleInfo.VIN {
			isPresent = true
		}
	}

	if !isPresent {
		return fmt.Errorf("There is no car with VIN %v connected to account", VIN)
	}
	return nil
}
