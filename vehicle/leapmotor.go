package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/leapmotor"
	"golang.org/x/sync/errgroup"
)

const (
	leapmotorCertURL   = "https://raw.githubusercontent.com/markoceri/leapmotor-certs/main/app.crt"
	leapmotorKeyURL    = "https://raw.githubusercontent.com/markoceri/leapmotor-certs/main/app.key"
	leapmotorDBKeyCert = "leapmotor.app.cert"
	leapmotorDBKeyKey  = "leapmotor.app.key"
)

// Leapmotor is an api.Vehicle implementation for Leapmotor cars.
type Leapmotor struct {
	*embed
	*leapmotor.Provider
}

// SinglePhaseLeapmotor wraps Leapmotor to report 1-phase charging (e.g. T03).
type SinglePhaseLeapmotor struct {
	*Leapmotor
}

func (o SinglePhaseLeapmotor) Phases() int {
	return 1
}

func init() {
	registry.Add("leapmotor", NewLeapmotorFromConfig)
}

func getAppCertFromDB() (certPEM, keyPEM []byte, err error) {
	cert, err1 := settings.String(leapmotorDBKeyCert)
	key, err2 := settings.String(leapmotorDBKeyKey)
	if err1 != nil || err2 != nil {
		return nil, nil, fmt.Errorf("not cached")
	}
	return []byte(cert), []byte(key), nil
}

func updateAppCertInDB(log *util.Logger) (certPEM, keyPEM []byte, err error) {
	client := request.NewHelper(log)
	eg := new(errgroup.Group)
	eg.Go(func() error {
		var e error
		certPEM, e = client.GetBody(leapmotorCertURL)
		return e
	})
	eg.Go(func() error {
		var e error
		keyPEM, e = client.GetBody(leapmotorKeyURL)
		return e
	})
	if err = eg.Wait(); err != nil {
		return nil, nil, err
	}
	settings.SetString(leapmotorDBKeyCert, string(certPEM))
	settings.SetString(leapmotorDBKeyKey, string(keyPEM))
	return certPEM, keyPEM, nil
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

	certPEM, keyPEM, err := getAppCertFromDB()
	if err != nil {
		if certPEM, keyPEM, err = updateAppCertInDB(log); err != nil {
			return nil, fmt.Errorf("leapmotor: fetch app certs: %w", err)
		}
	}

	newIdentity := func(certPEM, keyPEM []byte) (*leapmotor.Identity, error) {
		return leapmotor.NewIdentity(log, certPEM, keyPEM, cc.User, cc.Password)
	}

	identity, err := newIdentity(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	if err := identity.TryRestore(); err != nil {
		if loginErr := identity.Login(); loginErr != nil {
			if strings.Contains(loginErr.Error(), "tls") {
				// App cert may be stale — refresh and retry once
				if certPEM, keyPEM, err = updateAppCertInDB(log); err != nil {
					return nil, fmt.Errorf("leapmotor: refresh app certs: %w", err)
				}
				if identity, err = newIdentity(certPEM, keyPEM); err != nil {
					return nil, err
				}
				if err = identity.Login(); err != nil {
					return nil, err
				}
			} else {
				return nil, loginErr
			}
		}
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

	lm := &Leapmotor{
		embed:    &cc.embed,
		Provider: leapmotor.NewProvider(api, matched.VIN, matched.CarType, cc.Cache),
	}
	if matched.CarType == "T03" {
		return SinglePhaseLeapmotor{lm}, nil
	}
	return lm, nil
}
