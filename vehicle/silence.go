package vehicle

import (
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
	apiG func() (silence.Vehicle, error)
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
		return nil, api.ErrMissingCredentials
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
		v.apiG = provider.Cached(func() (silence.Vehicle, error) {
			return api.Status(vin)
		}, cc.Cache)
	}

	return v, err
}

// Soc implements the api.Vehicle interface
func (v *Silence) Soc() (float64, error) {
	res, err := v.apiG()
	return float64(res.BatterySoc), err
}

var _ api.VehicleRange = (*Silence)(nil)

// Range implements the api.VehicleRange interface
func (v *Silence) Range() (int64, error) {
	res, err := v.apiG()
	return int64(res.Range), err
}

var _ api.VehicleOdometer = (*Silence)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Silence) Odometer() (float64, error) {
	res, err := v.apiG()
	return float64(res.Odometer), err
}
