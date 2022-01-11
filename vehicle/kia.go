package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bluelink"
)

// Kia is an api.Vehicle implementation
type Kia struct {
	*embed
	*bluelink.Provider // provides the api implementations
}

func init() {
	registry.Add("kia", NewKiaFromConfig)
}

// NewKiaFromConfig creates a new Vehicle
func NewKiaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Expiry: expiry,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	settings := bluelink.Config{
		URI:               "https://prd.eu-ccapi.kia.com:8080",
		BasicToken:        "ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		CCSPServiceID:     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
		CCSPApplicationID: bluelink.KiaAppID,
		AuthClientID:      "572e0304-5f8d-4b4c-9dd5-41aa84eed160",
		BrandAuthUrl:      "https://eu-account.kia.com/auth/realms/eukiaidm/protocol/openid-connect/auth?client_id=%s&scope=openid%%20profile%%20email%%20phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
	}

	log := util.NewLogger("kia").Redact(cc.User, cc.Password, cc.VIN)
	identity := bluelink.NewIdentity(log, settings)

	if err := identity.Login(cc.User, cc.Password); err != nil {
		return nil, err
	}

	api := bluelink.NewAPI(log, settings.URI, identity, cc.Cache)

	vehicles, err := api.Vehicles()
	if err != nil {
		return nil, err
	}

	var vehicle bluelink.Vehicle
	if cc.VIN == "" && len(vehicles) == 1 {
		vehicle = vehicles[0]
		log.DEBUG.Printf("found vehicle: %v", cc.VIN)
	} else {
		for _, v := range vehicles {
			if v.VIN == strings.ToUpper(cc.VIN) {
				vehicle = v
			}
		}
	}

	if len(vehicle.VIN) == 0 {
		return nil, fmt.Errorf("cannot find vehicle: %v", vehicles)
	}

	v := &Kia{
		embed:    &cc.embed,
		Provider: bluelink.NewProvider(api, vehicle.VehicleID, cc.Expiry, cc.Cache),
	}

	return v, nil
}
