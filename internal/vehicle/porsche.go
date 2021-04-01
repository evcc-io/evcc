package vehicle

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/porsche"
	"github.com/andig/evcc/util"
)

const (
	porscheAPIClientID          = "4mPO3OE5Srjb1iaUGWsbqKBvvesya8oA"
	porscheEmobilityAPIClientID = "gZLSI7ThXFB4d2ld9t8Cx2DBRvGr1zN2"
)

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	*porsche.Provider // provides the api implementations
}

func init() {
	registry.Add("porsche", NewPorscheFromConfig)
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	log := util.NewLogger("porsche")

	var err error

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Porsche{
		embed: &embed{cc.Title, cc.Capacity},
	}

	api := porsche.NewAPI(log, porscheAPIClientID, porscheEmobilityAPIClientID, cc.User, cc.Password)
	err = api.Login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	vin, err := api.FindVehicle(cc.VIN)

	v.Provider = porsche.NewProvider(api, vin, cc.Cache)

	return v, err
}
