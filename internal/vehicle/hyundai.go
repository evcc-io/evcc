package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/bluelink"
	"github.com/andig/evcc/util"
)

// Hyundai is an api.Vehicle implementation
type Hyundai struct {
	*embed
	*bluelink.API
}

func init() {
	registry.Add("hyundai", NewHyundaiFromConfig)
}

// NewHyundaiFromConfig creates a new Vehicle
func NewHyundaiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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
		URI:               "https://prd.eu-ccapi.hyundai.com:8080",
		BasicToken:        "NmQ0NzdjMzgtM2NhNC00Y2YzLTk1NTctMmExOTI5YTk0NjU0OktVeTQ5WHhQekxwTHVvSzB4aEJDNzdXNlZYaG10UVI5aVFobUlGampvWTRJcHhzVg==",
		CCSPServiceID:     "6d477c38-3ca4-4cf3-9557-2a1929a94654",
		CCSPApplicationID: "99cfff84-f4e2-4be8-a5ed-e5b755eb6581",
		BrandAuthUrl:      "https://eu-account.hyundai.com/auth/realms/euhyundaiidm/protocol/openid-connect/auth?client_id=97516a3c-2060-48b4-98cd-8e7dcd3c47b2&scope=openid%%20profile%%20email%%20phone&response_type=code&hkid_session_reset=true&redirect_uri=%s/api/v1/user/integration/redirect/login&ui_locales=%s&state=%s:%s",
	}

	log := util.NewLogger("hyundai")
	identity, err := bluelink.NewIdentity(log, settings)
	if err != nil {
		return nil, err
	}

	if err := identity.Login(cc.User, cc.Password); err != nil {
		return nil, err
	}

	api := bluelink.NewAPI(log, identity, cc.Cache)

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

	v := &Hyundai{
		embed: &embed{cc.Title, cc.Capacity},
		API:   api,
	}

	return v, nil
}
