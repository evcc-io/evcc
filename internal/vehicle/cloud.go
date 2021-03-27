package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util/request"
)

// Cloud is an api.Vehicle implementation for Cloud cars
type Cloud struct {
	*embed
	*request.Helper
	user, password, vin string
	token               string
	tokenValid          time.Time
	chargeStateG        func() (float64, error)
}

func init() {
	registry.Add("cloud", NewCloudFromConfig)
}

// NewCloudFromConfig creates a new vehicle
func NewCloudFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title    string
		Capacity int64
		Brand    string
		Other    map[string]string `mapstructure:",remain"`
		Cache    time.Duration
	}{
		Cache: interval,
	}

	v := &Cloud{
		embed: &embed{cc.Title, cc.Capacity},
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v, nil
}

// chargeState implements the api.Vehicle interface
func (v *Cloud) chargeState() (float64, error) {
	return 0, api.ErrNotAvailable
}

// SoC implements the api.Vehicle interface
func (v *Cloud) SoC() (float64, error) {
	return v.chargeStateG()
}
