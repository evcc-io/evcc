package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/mystrom"
	"github.com/evcc-io/evcc/util"
)

// myStrom switch:
// https://api.mystrom.ch/#fbb2c698-e37a-4584-9324-3f8b2f615fe2

func init() {
	registry.Add("mystrom", NewMyStromFromConfig)
}

// NewMyStromFromConfig creates a myStrom meter from generic config
func NewMyStromFromConfig(other map[string]any) (api.Meter, error) {
	var cc struct {
		URI string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return mystrom.NewConnection(cc.URI), nil
}
