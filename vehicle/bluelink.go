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
func NewHyundaiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return newBluelinkFromConfig("hyundai", other)
}

// NewKiaFromConfig creates a new vehicle
func NewKiaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return newBluelinkFromConfig("kia", other)
}

// newBluelinkFromConfig creates a new Vehicle
func newBluelinkFromConfig(brand string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Language       string
		Region         string
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Language: "en",
		Region:   "EU",
		Expiry:   expiry,
		Cache:    interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	settings, err := bluelink.GetRegionSettings(brand, cc.Region)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)
	identity := bluelink.NewIdentity(log, settings)

	if err := identity.Login(cc.User, cc.Password, cc.Language); err != nil {
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

