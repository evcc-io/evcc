package vehicle

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/id"
	"github.com/andig/evcc/vehicle/vw"
)

// https://github.com/TA2k/ioBroker.vw-connect

// ID is an api.Vehicle implementation for ID cars
type ID struct {
	*embed
	*id.Provider // provides the api implementations
}

func init() {
	registry.Add("id", NewIDFromConfig)
}

// NewIDFromConfig creates a new vehicle
func NewIDFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &ID{
		embed: &embed{cc.Title, cc.Capacity},
	}

	log := util.NewLogger("id")
	identity := vw.NewIdentity(log, "")

	query := url.Values(map[string][]string{
		"response_type": {"code id_token token"},
		"client_id":     {"a24fba63-34b3-4d43-b181-942111e6bda8@apps_vw-dilab_com"},
		"redirect_uri":  {"weconnect://authenticated"},
		"scope":         {"openid profile badge cars dealers vin"},
	})

	err := identity.Login(query, cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := id.NewAPI(log, identity)

	if cc.VIN == "" {
		cc.VIN, err = findVehicle(api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", cc.VIN)
		}
	}

	v.Provider = id.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)

	return v, err
}
