package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tplink"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("tplink", NewTPLinkFromConfig)
}

// NewTPLinkFromConfig creates a tapo meter from generic config
func NewTPLinkFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	var cc struct {
		URI string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return tplink.NewConnection(cc.URI)
}
