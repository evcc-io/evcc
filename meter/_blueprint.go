package meter

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Blueprint meter implementation
type Blueprint struct {
	*request.Helper
	cache time.Duration
}

func init() {
	// registry.Add("foo", NewBlueprintFromConfig)
}

// NewBlueprintFromConfig creates a blueprint meter from generic config
func NewBlueprintFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		URI   string
		Cache time.Duration
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewBlueprint(cc.URI, cc.Cache)
}

// NewBlueprint creates Blueprint charger
func NewBlueprint(uri string, cache time.Duration) (api.Meter, error) {
	log := util.NewLogger("foo")

	m := &Blueprint{
		Helper: request.NewHelper(log),
		cache:  cache,
	}

	return m, nil
}

// CurrentPower implements the api.Meter interface
func (m *Blueprint) CurrentPower() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.MeterEnergy = (*Blueprint)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *Blueprint) TotalEnergy() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.PhaseCurrents = (*Blueprint)(nil)

// Currents implements the api.PhaseCurrents interface
func (m *Blueprint) Currents() (float64, float64, float64, error) {
	return 0, 0, 0, api.ErrNotAvailable
}
