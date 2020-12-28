package vehicle

import (
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/vw"
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
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Skoda{
		embed: &embed{cc.Title, cc.Capacity},
	}

	log := util.NewLogger("skoda")
	identity := vw.NewIdentity(log, "28cd30c6-dee7-4529-a0e6-b1e07ff90b79")

	query := url.Values(map[string][]string{
		"response_type": {"code id_token"},
		"client_id":     {"7f045eee-7003-4379-9968-9355ed2adb06%40apps_vw-dilab_com"},
		"redirect_uri":  {"skodaconnect://oidc.login/"},
		"scope":         {"openid profile phone address cars email birthdate badge dealers driversLicense mbb"},
	})

	err := identity.Login(query, cc.User, cc.Password)
	if err == nil {
		api := vw.NewAPI(log, identity, "VW", "CZ")

		if cc.VIN == "" {
			cc.VIN, err = findVehicle(api.Vehicles())
			if err == nil {
				log.DEBUG.Printf("found vehicle: %v", cc.VIN)
			}
		}

		v.Provider = vw.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)
	}

	return v, err
}
