package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/id"
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

	v := &ID{
		embed: &cc.embed,
	}

	log := util.NewLogger("id").Redact(cc.User, cc.Password, cc.VIN)

	ts := id.NewIdentity(log, cc.User, cc.Password)
	err := ts.Login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := id.NewAPI(log, ts)
	api.Client.Timeout = cc.Timeout

	if cc.VIN == "" {
		cc.VIN, err = findVehicle(api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", cc.VIN)
		}
	}

	v.Provider = id.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)

	return v, err
}
