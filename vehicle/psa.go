package vehicle

import (
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/vehicle/psa"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.psa

func init() {
	registry.Add("citroen", func(other map[string]any) (api.Vehicle, error) {
		return newPSA("citroen", "clientsB2CCitroen", other)
	})
	registry.Add("ds", func(other map[string]any) (api.Vehicle, error) {
		return newPSA("ds", "clientsB2CDS", other)
	})
	registry.Add("opel", func(other map[string]any) (api.Vehicle, error) {
		return newPSA("opel", "clientsB2COpel", other)
	})
	registry.Add("peugeot", func(other map[string]any) (api.Vehicle, error) {
		return newPSA("peugeot", "clientsB2CPeugeot", other)
	})
}

// PSA is an api.Vehicle implementation for PSA cars
type PSA struct {
	*embed
	*psa.Provider // provides the api implementations
}

// newPSA creates a new vehicle
func newPSA(brand, realm string, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		VIN      string
		User     string
		Password string `mapstructure:"password"`
		Country  string
		Tokens   oauth.Tokens
		Cache    time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger(brand)
	log.Redact(cc.User, cc.Tokens.Access, cc.Tokens.Refresh)

	// optional seed token from `evcc token` (login happens in the ui)
	var seed *oauth2.Token
	if token, err := cc.Tokens.Token(); err == nil {
		seed = token
	}

	identity, err := psa.NewIdentity(log, brand, cc.User, strings.ToLower(cc.Country), seed)
	if err != nil {
		return nil, err
	}

	// TODO still needed?
	api := psa.NewAPI(log, identity, realm, identity.ClientID())

	// resolved on first use so the vehicle can be created before login
	var vid string
	resolve := func() (string, error) {
		if vid != "" {
			return vid, nil
		}

		vehicle, err := ensureVehicleEx(
			cc.VIN, api.Vehicles,
			func(v psa.Vehicle) (string, error) {
				return v.VIN, nil
			},
		)
		if err != nil {
			return "", err
		}

		vid = vehicle.ID

		return vid, nil
	}

	v := &PSA{
		embed:    &cc.embed,
		Provider: psa.NewProvider(api, resolve, cc.Cache),
	}

	return v, nil
}
