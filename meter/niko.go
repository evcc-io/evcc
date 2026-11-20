package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/niko"
	"github.com/evcc-io/evcc/util"
)

// Niko Home Control smart plug via Hobby API

func init() {
	registry.AddCtx("niko", NewNikoFromConfig)
}

// NewNikoFromConfig creates a Niko Home Control smart plug meter from generic config
func NewNikoFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		Host   string
		User   string
		Token  string
		Device string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return niko.NewConnection(ctx, cc.Host, cc.User, cc.Token, cc.Device)
}
