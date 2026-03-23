package vehicle

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bluelink_us"
)

type BluelinkUS struct {
	*embed
	*bluelink_us.Provider
}

func init() {
	registry.Add("hyundai-us", NewHyundaiUSFromConfig)
}

func NewHyundaiUSFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		Pin            string
		VIN            string
		Cache          time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}
	if cc.Pin == "" {
		return nil, errors.New("PIN is required")
	}

	log := util.NewLogger("hyundai-us").Redact(cc.User, cc.Password, cc.VIN, cc.Pin)

	identity := bluelink_us.NewIdentity(log, cc.User, cc.Password)
	if err := identity.Login(); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	// Create temporary API to fetch vehicles (without vehicle-specific headers)
	tempCfg := bluelink_us.APIConfig{
		User: cc.User,
	}
	tempAPI := bluelink_us.NewAPI(log, identity, tempCfg)
	vehicles, err := tempAPI.Vehicles()
	if err != nil {
		return nil, fmt.Errorf("cannot get vehicles: %w", err)
	}

	// Find matching vehicle by VIN
	vehicle, err := ensureVehicleEx(
		cc.VIN,
		func() ([]bluelink_us.Vehicle, error) { return vehicles, nil },
		func(v bluelink_us.Vehicle) (string, error) { return v.VIN, nil },
	)
	if err != nil {
		return nil, err
	}

	// Create full API with vehicle-specific headers
	apiCfg := bluelink_us.APIConfig{
		User:           cc.User,
		Pin:            cc.Pin,
		RegistrationID: vehicle.RegID,
		VIN:            vehicle.VIN,
		Generation:     vehicle.VehicleGeneration,
	}
	vehicleAPI := bluelink_us.NewAPI(log, identity, apiCfg)

	v := &BluelinkUS{
		embed:    &cc.embed,
		Provider: bluelink_us.NewProvider(vehicleAPI, cc.Cache),
	}

	if v.Title_ == "" {
		v.Title_ = vehicle.NickName
	}

	return v, nil
}
