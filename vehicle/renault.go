package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/renault"
	"github.com/evcc-io/evcc/vehicle/renault/gigya"
	"github.com/evcc-io/evcc/vehicle/renault/kamereon"
	"github.com/evcc-io/evcc/vehicle/renault/keys"
)

// Credits to
//  https://github.com/hacf-fr/renault-api
//  https://github.com/edent/Renault-Zoe-API/issues/18
//  https://github.com/epenet/Renault-Zoe-API/blob/newapimockup/Test/MyRenault.py
//  https://github.com/jamesremuscat/pyze
//  https://muscatoxblog.blogspot.com/2019/07/delving-into-renaults-new-api.html
//  https://renault-api.readthedocs.io/en/latest/index.html

// Renault is an api.Vehicle implementation for Renault cars
type Renault struct {
	*embed
	*renault.Provider
}

func init() {
	registry.AddCtx("dacia", func(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
		return NewRenaultDaciaFromConfig(ctx, "dacia", other)
	})
	registry.AddCtx("renault", func(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
		return NewRenaultDaciaFromConfig(ctx, "renault", other)
	})
}

// NewRenaultDaciaFromConfig creates a new Renault/Dacia vehicle
func NewRenaultDaciaFromConfig(ctx context.Context, brand string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                       `mapstructure:",squash"`
		User, Password, Region, VIN string
		WakeupMode                  string
		Cache                       time.Duration
		Timeout                     time.Duration
	}{
		Region:     "de_DE",
		WakeupMode: "default",
		Cache:      interval,
		Timeout:    request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)

	v := &Renault{
		embed: cc.embed.withContext(ctx),
	}

	keys := keys.New(log)
	keys.Load(cc.Region)

	identity := gigya.NewIdentity(log, keys.Gigya)
	if err := identity.Login(cc.User, cc.Password); err != nil {
		return nil, err
	}

	api := kamereon.New(log, keys.Kamereon, identity, func() error {
		return identity.Login(cc.User, cc.Password)
	})
	api.Client.Timeout = cc.Timeout

	accountID, err := api.Person(identity.PersonID, brand)
	if err != nil {
		return nil, err
	}

	vehicle, err := ensureVehicleEx(cc.VIN,
		func() ([]kamereon.Vehicle, error) {
			return api.Vehicles(accountID)
		},
		func(v kamereon.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)

	if err == nil {
		err = vehicle.Available()
	}

	v.Provider = renault.NewProvider(api, accountID, vehicle.VIN, cc.WakeupMode, cc.Cache)

	return v, err
}
