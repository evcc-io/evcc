package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tapo"
	"github.com/evcc-io/evcc/util"
)

// TP-Link Tapo meter implementation
func init() {
	registry.Add("tapo", NewTapoFromConfig)
}

// NewTapoFromConfig creates a tapo meter from generic config
func NewTapoFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		URI, User, Password string `validate:"required"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return tapo.NewConnection(cc.URI, cc.User, cc.Password)
}
