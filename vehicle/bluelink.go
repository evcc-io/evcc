package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bluelink"
)

// https://github.com/Hacksore/bluelinky
// https://github.com/Hyundai-Kia-Connect/hyundai_kia_connect_api/pull/353/files

// Bluelink is an api.Vehicle implementation
type Bluelink struct {
	*embed
	*bluelink.Provider
}

func init() {
	registry.Add("kia", NewKiaFromConfig)
	registry.Add("hyundai", NewHyundaiFromConfig)
}

// NewHyundaiFromConfig creates a new vehicle
func NewHyundaiFromConfig(other map[string]any) (api.Vehicle, error) {
	settings := bluelink.Config{
		URI:               "https://prd.eu-ccapi.hyundai.com:8080",
		CCSPServiceID:     "6d477c38-3ca4-4cf3-9557-2a1929a94654",
		CCSPServiceSecret: "KUy49XxPzLpLuoK0xhBC77W6VXhmtQR9iQhmIFjjoY4IpxsV",
		CCSPApplicationID: "014d2225-8495-4735-812d-2616334fd15d",
		Cfb:               "RFtoRq/vDXJmRndoZaZQyfOot7OrIqGVFj96iY2WL3yyH5Z/pUvlUhqmCxD2t+D65SQ=",
		BasicToken:        "NmQ0NzdjMzgtM2NhNC00Y2YzLTk1NTctMmExOTI5YTk0NjU0OktVeTQ5WHhQekxwTHVvSzB4aEJDNzdXNlZYaG10UVI5aVFobUlGampvWTRJcHhzVg==",
		PushType:          "GCM",
		LoginFormHost:     "https://idpconnect-eu.hyundai.com",
		Brand:             "hyundai",
	}

	return newBluelinkFromConfig("hyundai", other, settings)
}

// NewKiaFromConfig creates a new vehicle
func NewKiaFromConfig(other map[string]any) (api.Vehicle, error) {
	settings := bluelink.Config{
		URI:               "https://prd.eu-ccapi.kia.com:8080",
		CCSPServiceID:     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
		CCSPServiceSecret: "secret",
		CCSPApplicationID: "a2b8469b-30a3-4361-8e13-6fceea8fbe74",
		Cfb:               "wLTVxwidmH8CfJYBWSnHD6E0huk0ozdiuygB4hLkM5XCgzAL1Dk5sE36d/bx5PFMbZs=",
		BasicToken:        "ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		LoginFormHost:     "https://idpconnect-eu.kia.com",
		PushType:          "APNS",
		Brand:             "kia",
	}

	return newBluelinkFromConfig("kia", other, settings)
}

// newBluelinkFromConfig creates a new Vehicle
func newBluelinkFromConfig(brand string, other map[string]any, settings bluelink.Config) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Language       string
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Language: "en",
		Expiry:   expiry,
		Cache:    interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)
	identity := bluelink.NewIdentity(log, settings)

	if err := identity.Login(cc.User, cc.Password, cc.Language, brand); err != nil {
		return nil, err
	}

	api := bluelink.NewAPI(log, settings.URI, identity.Request)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v bluelink.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)
	if err != nil {
		return nil, err
	}

	v := &Bluelink{
		embed:    &cc.embed,
		Provider: bluelink.NewProvider(api, vehicle, cc.Expiry, cc.Cache),
	}

	return v, nil
}
