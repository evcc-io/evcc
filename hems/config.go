package hems

import (
	"context"
	"errors"
	"strings"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/eebus"
	"github.com/evcc-io/evcc/hems/relay"
)

// HEMS describes the HEMS system interface
type HEMS interface {
	Run()
}

// NewFromConfig creates new HEMS from config
func NewFromConfig(ctx context.Context, typ string, other map[string]any, site site.API) (HEMS, error) {
	switch strings.ToLower(typ) {
	case "sma", "shm", "semp":
		return nil, errors.New("breaking change: Sunny Home Manager integration is always on. See https://github.com/evcc-io/evcc/releases and https://docs.evcc.io/en/docs/integrations/sma-sunny-home-manager")
	case "eebus":
		return eebus.NewFromConfig(ctx, other, site)
	case "relay":
		return relay.NewFromConfig(ctx, other, site)
	default:
		return nil, errors.New("unknown hems: " + typ)
	}
}
