package vehicle

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/vw"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("vw", NewVWFromConfig)
}

// NewVWFromConfig creates a new vehicle
func NewVWFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &VW{
		embed: &embed{cc.Title, cc.Capacity},
	}

	log := util.NewLogger("vw")
	identity := vw.NewIdentity(log, "38761134-34d0-41f3-9a73-c4be88d7d337")

	query := url.Values(map[string][]string{
		"response_type": {"id_token token"},
		"client_id":     {"9496332b-ea03-4091-a224-8c746b885068@apps_vw-dilab_com"},
		"redirect_uri":  {"carnet://identity-kit/login"},
		"scope":         {"openid profile mbb cars birthdate nickname address phone"},
	})

	err := identity.Login(query, cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, "VW", "DE")

	if cc.VIN == "" {
		cc.VIN, err = findVehicle(api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", cc.VIN)
		}
	}

	v.Provider = vw.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)

	return v, err
}
