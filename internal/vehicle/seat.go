package vehicle

import (
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/vw"
	"github.com/andig/evcc/util"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// Seat is an api.Vehicle implementation for Seat cars
type Seat struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("seat", NewSeatFromConfig)
}

// NewSeatFromConfig creates a new vehicle
func NewSeatFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Seat{
		embed: &cc.embed,
	}

	log := util.NewLogger("seat")
	identity := vw.NewIdentity(log, "9dcc70f0-8e79-423a-a3fa-4065d99088b4")

	query := url.Values(map[string][]string{
		"response_type": {"code id_token"},
		"client_id":     {"50f215ac-4444-4230-9fb1-fe15cd1a9bcc@apps_vw-dilab_com"},
		"redirect_uri":  {"seatconnect://identity-kit/login"},
		"scope":         {"openid profile mbb cars birthdate nickname address phone"},
	})

	err := identity.Login(query, cc.User, cc.Password)
	if err == nil {
		api := vw.NewAPI(log, identity, "VW", "ES")

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
