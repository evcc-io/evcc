package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bmw/cardata"
)

// Cardata is an api.Vehicle implementation for BMW and Mini cars
type Cardata struct {
	*embed
	*cardata.Provider // provides the api implementations
}

func init() {
	registry.AddCtx("cardata", NewCardataFromConfig)
}

// NewCardataFromConfig creates a new BMW/Mini CarData vehicle
func NewCardataFromConfig(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed         `mapstructure:",squash"`
		ClientID, VIN string
		Cache         time.Duration
	}{
		Cache: 30 * time.Minute, // 50 requests per day
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" {
		return nil, api.ErrMissingCredentials
	}

	config := cardata.Config
	config.ClientID = cc.ClientID

	v := &Cardata{
		embed: &cc.embed,
	}

	log := util.NewLogger("cardata").Redact(cc.ClientID)
	ts, err := cardata.NewIdentity(ctx, log, &config)
	if err != nil {
		return nil, err
	}

	api := cardata.NewAPI(log, ts)

	vehicle, err := ensureVehicle(
		cc.VIN, api.Vehicles,
	)

	container, err := api.EnsureContainer()
	if err != nil {
		return nil, err
	}

	v.Provider = cardata.NewProvider(log, api, ts, vehicle, container)

	return v, nil
}
