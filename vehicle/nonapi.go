package vehicle

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// NonAPI is an api.Vehicle implementation for e-vehicles having no api
type NonAPI struct {
	*embed
	*request.Helper
	vin          string
	chargeStateG func() (float64, error)
}

func init() {
	registry.Add("nonapi", NewNonAPIFromConfig)
}

// NewNonAPIFromConfig creates a new vehicle
func NewNonAPIFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title    string
		Capacity int64
		VIN      string
		Cache    time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("nonapi")

	v := &NonAPI{
		embed:  &embed{cc.Title, cc.Capacity},
		Helper: request.NewHelper(log),
		vin:    strings.ToUpper(cc.VIN),
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v, nil
}

// chargeState implements the api.Vehicle interface
func (v *NonAPI) chargeState() (float64, error) {
	// Non
	return 0, nil
}

// SoC implements the api.Vehicle interface
func (v *NonAPI) SoC() (float64, error) {
	return 0, nil
}
