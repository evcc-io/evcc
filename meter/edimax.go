package meter

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/edimax"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("edimax", NewEdimaxFromConfig)
}

// NewEdimaxFromConfig creates an Edimax meter from generic config
func NewEdimaxFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		Host     string
		User     string
		Password string
		Cache    time.Duration
	}{
		User:     "admin",
		Password: "1234",
		Cache:    time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return edimax.NewConnection(ctx, cc.Host, cc.User, cc.Password, cc.Cache), nil
}
