package vehicle

import (
	"fmt"
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
	// sru_250814: this was good enough for when we only had to consider EU
	//	but needs refactoring if other regions are to be supported. The general
	//	question here is: let the `NewIdentity` decide about the values by using
	//  the region information or use the region information first to call
	//	call different identity providers per region?
	//  Gut feeling is to go with the second approach. But we have to keep the
	//	different pseudo inits to be able to inject the brand into the config
	//	struct.
	registry.Add("kia", NewKiaFromConfig)
	registry.Add("hyundai", NewHyundaiFromConfig)
	registry.Add("genesis", NewGenesisFromConfig)
}

// NewHyundaiFromConfig creates a new vehicle
func NewHyundaiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return newBluelinkFromConfig("hyundai", other)
}

// NewKiaFromConfig creates a new vehicle
func NewKiaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return newBluelinkFromConfig("kia", other)
}

// NewGenesisConfig creates a new vehicle
func NewGenesisFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return newBluelinkFromConfig("kia", other)
}

// newBluelinkFromConfig creates a new Vehicle
func newBluelinkFromConfig(brand string, other map[string]interface{}) (api.Vehicle, error) {
	// TODO: investigate why mapping of `template` suddenly fails.
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Language       string
		Region         string
		Brand          string `mapstructure:"template"`
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Language: "en",
		// default for now, remove once there are more supported regions?
		// might also work as fallback for vehicles created when there was
		// no region differentiation
		Region: bluelink.RegionEurope,
		Expiry: expiry,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// sru_250808: seems like we're suddenly missing `template` from `other`
	// 	 but for now I'd like to carry the brand with the config
	if cc.Brand == "" {
		cc.Brand = brand
	}

	// check whether we have a base config for given region/brand
	if bluelink.ConfigMap[cc.Region][brand] == nil {
		return nil, fmt.Errorf("no config map for brand %s in region %s", brand, cc.Region)
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)
	// sru_250808: debug only, remove or TRACE for production
	log.INFO.Printf("Other: %v", other)
	log.INFO.Printf("CC: %v\n", cc)

	// Decide what region to create the API for
	switch cc.Region {
	case bluelink.RegionAustralia:
		settings, err := bluelink.PopulateSettingsAU(cc.Brand, cc.Region)
		if err != nil {
			return nil, err
		}
		log.INFO.Printf("Got %s/%s settings:\n%v", cc.Brand, cc.Region, settings)

		identity := bluelink.NewIdentity(log, settings)
		if err := identity.LoginAU(cc.User, cc.Password); err != nil {
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
	case bluelink.RegionEurope:
		settings, err := bluelink.PopulateSettingsEU(cc.Brand, cc.Region)
		if err != nil {
			return nil, err
		}

		identity := bluelink.NewIdentity(log, settings)
		if err := identity.LoginEU(cc.User, cc.Password, cc.Language, cc.Brand); err != nil {
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

	return nil, fmt.Errorf("unsupported region: %s", cc.Region)
}
