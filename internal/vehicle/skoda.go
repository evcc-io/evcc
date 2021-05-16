package vehicle

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/vw"
	"github.com/andig/evcc/util"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// Skoda is an api.Vehicle implementation for Skoda cars
type Skoda struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("skoda", NewSkodaFromConfig)
}

// NewSkodaFromConfig creates a new vehicle
func NewSkodaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Skoda{
		embed: &cc.embed,
	}

	log := util.NewLogger("skoda")
	identity := vw.NewIdentity(log)

	query := url.Values(map[string][]string{
		"response_type": {"code id_token"},
		"client_id":     {"7f045eee-7003-4379-9968-9355ed2adb06@apps_vw-dilab_com"},
		"redirect_uri":  {"skodaconnect://oidc.login/"},
		"scope":         {"openid profile phone address cars email birthdate badge dealers driversLicense mbb"},
	})

	err := identity.LoginVAG("28cd30c6-dee7-4529-a0e6-b1e07ff90b79", query, cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, "VW", "CZ")

	if cc.VIN == "" {
		cc.VIN, err = findVehicle(api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", cc.VIN)
		}
	}

	if err == nil {
		if err = api.HomeRegion(strings.ToUpper(cc.VIN)); err == nil {
			v.Provider = vw.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)
		}
	}

	return v, err
}
