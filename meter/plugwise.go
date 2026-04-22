package meter

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/plugwise"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("plugwise", NewPlugwiseFromConfig)
}

// NewPlugwiseFromConfig creates a Plugwise Smile P1 meter from a generic YAML config map.
// Required config keys: uri (device address), password (SmileID).
// Optional: cache (default 1s TTL for HTTP response coalescing).
func NewPlugwiseFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		Password string
		Cache    time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return plugwise.NewConnection(cc.URI, cc.Password, cc.Cache)
}
