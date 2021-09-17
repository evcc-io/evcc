package vehicle

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vw"
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
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
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
		"client_id":     {"f9a2359a-b776-46d9-bd0c-db1904343117@apps_vw-dilab_com"},
		"redirect_uri":  {"skodaconnect://oidc.login/"},
		"scope":         {"openid mbb profile"},
	})

	err := identity.LoginVAG("afb0473b-6d82-42b8-bfea-cead338c46ef", query, cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, "VW", "CZ")
	api.Client.Timeout = cc.Timeout

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
