package vehicle

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/myskoda"
)

// MySkoda is an api.Vehicle implementation for the MyŠkoda public api
type MySkoda struct {
	*embed
	*myskoda.Provider
}

func init() {
	registry.Add("myskoda", NewMySkodaFromConfig)
}

// NewMySkodaFromConfig creates a new vehicle
func NewMySkodaFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed   `mapstructure:",squash"`
		VIN     string
		ApiKey  string
		Sandbox bool
		Cache   time.Duration
		Timeout time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ApiKey == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.VIN == "" {
		return nil, errors.New("missing vin")
	}

	log := util.NewLogger("myskoda").Redact(cc.ApiKey, cc.VIN)

	uri := myskoda.BaseURI
	if cc.Sandbox {
		uri = myskoda.SandboxURI
	}

	apiC := myskoda.NewAPI(log, uri, cc.ApiKey)
	apiC.Client.Timeout = cc.Timeout

	// validate api key and vin, use vehicle name as title
	res, err := apiC.Vehicle(cc.VIN, "info")
	if err != nil {
		return nil, err
	}

	v := &MySkoda{
		embed: &cc.embed,
	}
	v.fromVehicle(res.Vehicle.Name, 0)

	v.Provider = myskoda.NewProvider(apiC, cc.VIN, cc.Cache)

	return v, nil
}
