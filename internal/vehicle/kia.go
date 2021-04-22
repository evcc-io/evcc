package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/bluelink"
	"github.com/andig/evcc/util"
)

// Kia is an api.Vehicle implementation
type Kia struct {
	*embed
	*bluelink.API
}

func init() {
	registry.Add("kia", NewKiaFromConfig)
}

// NewKiaFromConfig creates a new Vehicle
func NewKiaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title          string
		Capacity       int64
		User, Password string
		VIN            string
		Cache          time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("missing credentials")
	}

	settings := bluelink.Config{
		URI:               "https://prd.eu-ccapi.kia.com:8080",
		TokenAuth:         "ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		CCSPServiceID:     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
		CCSPApplicationID: "693a33fa-c117-43f2-ae3b-61a02d24f417",
		BrandAuthUrl:      "https://eu-account.kia.com/auth/realms/eukiaidm/protocol/openid-connect/auth?client_id=f4d531c7-1043-444d-b09a-ad24bd913dd4&scope=openid%%20profile%%20email%%20phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
	}

	log := util.NewLogger("kia")
	api, err := bluelink.New(log, cc.User, cc.Password, cc.Cache, settings)
	if err != nil {
		return nil, err
	}

	vehicles, err := api.Vehicles()
	if err != nil {
		return nil, err
	}

	if cc.VIN == "" && len(vehicles) == 1 {
		api.Vehicle = vehicles[0]
	} else {
		for _, vehicle := range vehicles {
			if vehicle.Vin == strings.ToUpper(cc.VIN) {
				api.Vehicle = vehicle
			}
		}
	}

	if len(api.Vehicle.Vin) == 0 {
		return nil, errors.New("vin not found")
	}

	v := &Kia{
		embed: &embed{cc.Title, cc.Capacity},
		API:   api,
	}

	return v, nil
}
