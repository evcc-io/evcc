package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/silence"
)

// Silence is an api.Vehicle implementation for Silence S01 vehicles
type Silence struct {
	*embed
	apiG func() (interface{}, error)
}

func init() {
	registry.Add("silence", NewSilenceFromConfig)
}

// NewFordFromConfig creates a new vehicle
func NewSilenceFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
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
		return nil, errors.New("missing user or password")
	}

	log := util.NewLogger("s01").Redact(cc.User, cc.Password)

	v := &Silence{
		embed: &cc.embed,
	}

	identity, err := silence.NewIdentity(log, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	api := silence.NewAPI(log, identity)
	vin, err := ensureVehicle(strings.ToLower(cc.VIN), api.Vehicles)

	if err == nil {
		v.apiG = provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cc.Cache).InterfaceGetter()
	}

	return v, err
}

// SoC implements the api.Vehicle interface
func (v *Silence) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(silence.Vehicle); err == nil && ok {
		return float64(res.BatterySoc), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Silence)(nil)

// Range implements the api.VehicleRange interface
func (v *Silence) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(silence.Vehicle); err == nil && ok {
		return int64(res.Range), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Silence)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Silence) Odometer() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(silence.Vehicle); err == nil && ok {
		return float64(res.Odometer), nil
	}

	return 0, err
}
