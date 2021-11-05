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

// https://github.com/davidgiga1993/AudiAPI
// https://github.com/TA2k/ioBroker.vw-connect

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*vw.Provider // provides the api implementations
	// audiProvider *audi.Provider
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Audi{
		embed: &cc.embed,
	}

	log := util.NewLogger("audi").Redact(cc.User, cc.Password, cc.VIN)
	identity := vw.NewIdentity(log)

	query := url.Values(map[string][]string{
		"response_type": {"id_token token"},
		"client_id":     {"09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"},
		"redirect_uri":  {"myaudi:///"},
		"scope":         {"openid profile mbb"}, // vin badge birthdate nickname email address phone name picture
		"prompt":        {"login"},
		"ui_locales":    {"de-DE"},
	})

	ts := vw.NewTokenSource(log, identity, "77869e21-e30a-4a92-b016-48ab7d3db1d8", query, cc.User, cc.Password)
	err := identity.Login(ts)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, "Audi", "DE")
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
