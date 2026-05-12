package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/leapmotor"
	"golang.org/x/sync/errgroup"
)

const (
	leapmotorCertURL = "https://raw.githubusercontent.com/markoceri/leapmotor-certs/main/app.crt"
	leapmotorKeyURL  = "https://raw.githubusercontent.com/markoceri/leapmotor-certs/main/app.key"
)

// Leapmotor is an api.Vehicle implementation for Leapmotor cars.
type Leapmotor struct {
	*embed
	*leapmotor.Provider
}

func init() {
	registry.Add("leapmotor", NewLeapmotorFromConfig)
}

// NewLeapmotorFromConfig creates a new Leapmotor vehicle from config.
func NewLeapmotorFromConfig(other map[string]any) (api.Vehicle, error) {
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

	log := util.NewLogger("leapmotor").Redact(cc.User, cc.Password, cc.VIN)

	client := request.NewHelper(log)
	var certPEM, keyPEM []byte
	eg := new(errgroup.Group)
	eg.Go(func() error {
		var err error
		certPEM, err = client.GetBody(leapmotorCertURL)
		return err
	})
	eg.Go(func() error {
		var err error
		keyPEM, err = client.GetBody(leapmotorKeyURL)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("leapmotor: fetch app certs: %w", err)
	}

	identity, err := leapmotor.NewIdentity(log, certPEM, keyPEM, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}
	if err := identity.Login(); err != nil {
		return nil, err
	}

	api := leapmotor.NewAPI(log, identity)

	vehicles, err := api.Vehicles()
	if err != nil {
		return nil, fmt.Errorf("leapmotor: get vehicles: %w", err)
	}
	if len(vehicles) == 0 {
		return nil, fmt.Errorf("leapmotor: no vehicles found on account")
	}

	var matched *leapmotor.Vehicle
	for i := range vehicles {
		v := &vehicles[i]
		if cc.VIN == "" || v.VIN == cc.VIN {
			matched = v
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("leapmotor: VIN %s not found on account", cc.VIN)
	}

	return &Leapmotor{
		embed:    &cc.embed,
		Provider: leapmotor.NewProvider(api, matched.VIN, matched.CarType, cc.Cache),
	}, nil
}
