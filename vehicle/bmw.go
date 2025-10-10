package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	bmw "github.com/evcc-io/evcc/vehicle/bmw/connected"
)

// BMW is an api.Vehicle implementation for BMW and Mini cars
type BMW struct {
	*embed
	*bmw.Provider // provides the api implementations
}

func init() {
	registry.AddCtx("bmw", NewBMWFromConfig)
	registry.AddCtx("mini", NewMiniFromConfig)
}

// NewBMWFromConfig creates a new vehicle
func NewBMWFromConfig(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
	return NewBMWMiniFromConfig(ctx, "bmw", other)
}

// NewMiniFromConfig creates a new vehicle
func NewMiniFromConfig(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
	return NewBMWMiniFromConfig(ctx, "mini", other)
}

// NewBMWMiniFromConfig creates a new vehicle
func NewBMWMiniFromConfig(ctx context.Context, brand string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Hcaptcha            string
		Region              string
		Cache               time.Duration
	}{
		Region: "EU",
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" || cc.Hcaptcha == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &BMW{
		embed: cc.embed.withContext(ctx),
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)
	identity := bmw.NewIdentity(log, cc.Region)

	ts, err := identity.Login(cc.User, cc.Password, cc.Hcaptcha)
	if err != nil {
		return nil, err
	}

	api := bmw.NewAPI(log, brand, cc.Region, ts)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v bmw.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)

	if err == nil {
		v.Provider = bmw.NewProvider(api, vehicle.VIN, cc.Cache)
	}

	return v, err
}
