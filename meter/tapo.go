package meter

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/tapo"
)

// TP-Link Tapo meter implementation
func init() {
	registry.Add("tapo", NewTapoFromConfig)
}

// NewTapoFromConfig creates a tapo meter from generic config
func NewTapoFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return tapo.NewConnection(cc.URI, cc.User, cc.Password)
}
