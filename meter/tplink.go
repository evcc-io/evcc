package meter

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tplink"
	"github.com/evcc-io/evcc/util"
)

// TP-Link meter implementation
func init() {
	registry.Add("tplink", NewTPLinkFromConfig)
}

// NewTPLinkFromConfig creates a tapo meter from generic config
func NewTPLinkFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return tplink.NewConnection(cc.URI)
}
