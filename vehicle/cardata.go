package vehicle

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bmw/cardata"
)

// Cardata is an api.Vehicle implementation for BMW and Mini cars
type Cardata struct {
	*embed
	*cardata.Provider // provides the api implementations
}

func init() {
	registry.AddCtx("cardata", NewCardataFromConfig)
}

// NewCardataFromConfig creates a new BMW/Mini CarData vehicle
func NewCardataFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	var cc struct {
		embed         `mapstructure:",squash"`
		ClientID, VIN string
		Cache         time.Duration // 50 requests per day
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.VIN == "" {
		return nil, errors.New("missing vin")
	}

	if cc.ClientID == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Cache == 0 {
		// for non-streaming use 15m, access controlled by loadpoint
		isStreaming := slices.Contains(cc.embed.Features(), api.Streaming)
		cc.Cache = map[bool]time.Duration{false: 15 * time.Minute, true: 30 * time.Minute}[isStreaming]
	}

	v := &Cardata{
		embed: &cc.embed,
	}

	log := util.NewLogger("cardata").Redact(cc.ClientID, cc.VIN)

	ts, err := cardata.NewOAuth(cc.ClientID, cc.embed.GetTitle())
	if err != nil {
		return nil, err
	}

	api := cardata.NewAPI(log, ts)

	v.Provider = cardata.NewProvider(ctx, log, api, ts, cc.ClientID, cc.VIN, cc.Cache)

	return v, nil
}
