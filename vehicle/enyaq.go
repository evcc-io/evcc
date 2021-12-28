package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/skoda"
)

// https://github.com/lendy007/skodaconnect

// Enyaq is an api.Vehicle implementation for Skoda Enyaq cars
type Enyaq struct {
	*embed
	*skoda.Provider // provides the api implementations
}

func init() {
	registry.Add("enyaq", NewEnyaqFromConfig)
}

// NewEnyaqFromConfig creates a new vehicle
func NewEnyaqFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Enyaq{
		embed: &cc.embed,
	}

	var err error
	log := util.NewLogger("enyaq").Redact(cc.User, cc.Password, cc.VIN)

	if cc.VIN == "" {
		ts := skoda.NewIdentity(log, skoda.AuthParams, cc.User, cc.Password)
		if err = ts.Login(); err != nil {
			return v, fmt.Errorf("login failed: %w", err)
		}

		api := skoda.NewAPI(log, ts)
		api.Client.Timeout = cc.Timeout

		cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)
	}

	if err == nil {
		ts := skoda.NewIdentity(log, skoda.ConnectAuthParams, cc.User, cc.Password)
		if err = ts.Login(); err != nil {
			return v, fmt.Errorf("login failed: %w", err)
		}

		api := skoda.NewAPI(log, ts)
		api.Client.Timeout = cc.Timeout

		v.Provider = skoda.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
