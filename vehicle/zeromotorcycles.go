package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/zero"
)

// MG is an api.Vehicle implementation for probably all SAIC cars
type ZeroMotorcycle struct {
	*embed
	*zero.Provider // provides the api implementations
}

func init() {
	registry.Add("zero", NewZeroFromConfig)
}

// NewBMWFromConfig creates a new vehicle
func NewZeroFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	var res *zero.API
	var err error
	var unitId string

	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err = util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("Zero").Redact(cc.User, cc.Password)

	if res, err = zero.NewAPI(log, cc.User, cc.Password); err != nil {
		return nil, err
	}

	if unitId, err = retrievedeviceId(res, cc.VIN); err != nil {
		return nil, err
	}

	v := &ZeroMotorcycle{
		embed:    &cc.embed,
		Provider: zero.NewProvider(res, unitId, cc.Cache),
	}

	return v, nil
}

func retrievedeviceId(v *zero.API, vin string) (string, error) {
	var res zero.UnitData
	var err error
	if res, err = v.Vehicles(); err != nil {
		return "", err
	}

	if vin == "" {
		return res[0].Unitnumber, nil
	}

	for _, unit := range res {
		if unit.Name == vin {
			return unit.Unitnumber, nil
		}
	}

	return "", fmt.Errorf("vin not found")
}
