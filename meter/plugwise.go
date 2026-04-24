package meter

import (
	"errors"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/plugwise"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("plugwise", NewPlugwiseFromConfig)
}

//go:generate go tool decorate -f decoratePlugwise -b *plugwise.Connection -r api.Meter -t api.PhaseVoltages,api.PhaseCurrents,api.PhasePowers

// NewPlugwiseFromConfig creates a Plugwise Smile P1 meter from a generic YAML config map.
// Required config keys: uri (device address), password (SmileID).
// Optional: cache (default 1s TTL for HTTP response coalescing).
//
// The factory probes c.PhaseVoltages() once immediately after the connection is
// established. If all three phase voltages come back non-zero (three-phase DSMR
// 5.0+ firmware), it activates the api.PhasePowers, api.PhaseVoltages and
// api.PhaseCurrents optional interfaces via the generated decoratePlugwise.
// Otherwise all three are passed as nil — the generated decorator returns the
// bare *plugwise.Connection, which only satisfies api.Meter. This matches the
// zero-config pattern used by meter/tasmota.go.
//
// Probe limitation: the probe runs ONCE at startup. A device that is unreachable
// or rebooting at startup will degrade to single-phase for the life of the
// process (no phase interfaces). Restart EVCC to re-probe.
func NewPlugwiseFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		Password string
		Usage    string
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

	c, err := plugwise.NewConnection(cc.URI, cc.Password, cc.Cache)
	if err != nil {
		return nil, err
	}

	// Verify connectivity and credentials before doing anything else.
	// A 401 response means the SmileID (password) is wrong — surface that
	// immediately so the UI shows "wrong password" instead of "status: successful
	// but no values". Other errors (timeout, connection refused) are transient and
	// tolerated: the device may simply be unreachable at startup.
	if _, probeErr := c.CurrentPower(); probeErr != nil {
		var se *request.StatusError
		if errors.As(probeErr, &se) && se.HasStatus(http.StatusUnauthorized) {
			return nil, errors.New("wrong password")
		}
	}

	// Probe for three-phase support. If all three phase voltages come back
	// non-zero, activate optional phase interfaces via the generated decorator.
	// Errors here are silently tolerated — the cached dataG response from the
	// CurrentPower probe above is reused, so a 401 would already have been caught.
	var (
		vol func() (float64, float64, float64, error)
		cur func() (float64, float64, float64, error)
		pow func() (float64, float64, float64, error)
	)
	if v1, v2, v3, err := c.PhaseVoltages(); err == nil && v1*v2*v3 > 0 {
		vol = c.PhaseVoltages
		cur = c.Currents
		pow = c.PhasePowers
	}

	return decoratePlugwise(c, vol, cur, pow), nil
}
