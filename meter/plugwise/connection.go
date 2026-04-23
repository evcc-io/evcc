package plugwise

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection holds the HTTP client and cached fetch function for a Plugwise Smile P1.
type Connection struct {
	*request.Helper
	uri   string
	dataG func() (DomainObjects, error)
}

// NewConnection creates a new Plugwise Smile P1 connection with BasicAuth transport and a cached fetch.
func NewConnection(uri, password string, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("plugwise")
	basicAuth := transport.BasicAuthHeader("smile", password)
	log.Redact(password, basicAuth)

	c := &Connection{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
	}
	c.Client.Transport = transport.BasicAuth("smile", password, c.Client.Transport)

	c.dataG = util.Cached(func() (DomainObjects, error) {
		var res DomainObjects
		req, err := request.New(http.MethodGet, c.uri+"/core/domain_objects", nil)
		if err != nil {
			return res, err
		}
		err = c.DoXML(req, &res)
		return res, err
	}, cache)

	return c, nil
}

var _ api.Meter = (*Connection)(nil)

// CurrentPower implements api.Meter. Returns signed net grid power in watts.
// Positive = grid import (consuming more than producing), negative = grid export.
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.dataG()
	if err != nil {
		return 0, err
	}
	consumed := res.Location.Logs.PowerWatts("electricity_consumed")
	produced := res.Location.Logs.PowerWatts("electricity_produced")
	return consumed - produced, nil
}

// PhasePowers returns signed net power per phase in watts.
// Positive = grid import on that phase, negative = export. Matches the
// convention of CurrentPower (D-02). Underlying per-phase XML fields:
// electricity_phase_{one,two,three}_{consumed,produced}.
func (c *Connection) PhasePowers() (float64, float64, float64, error) {
	res, err := c.dataG()
	if err != nil {
		return 0, 0, 0, err
	}
	l := res.Location.Logs
	p1 := l.PowerWatts("electricity_phase_one_consumed") - l.PowerWatts("electricity_phase_one_produced")
	p2 := l.PowerWatts("electricity_phase_two_consumed") - l.PowerWatts("electricity_phase_two_produced")
	p3 := l.PowerWatts("electricity_phase_three_consumed") - l.PowerWatts("electricity_phase_three_produced")
	return p1, p2, p3, nil
}

// PhaseVoltages returns per-phase voltage in volts.
// Underlying XML fields: voltage_phase_{one,two,three} (unit V).
func (c *Connection) PhaseVoltages() (float64, float64, float64, error) {
	res, err := c.dataG()
	if err != nil {
		return 0, 0, 0, err
	}
	l := res.Location.Logs
	return l.VoltageVolts("voltage_phase_one"),
		l.VoltageVolts("voltage_phase_two"),
		l.VoltageVolts("voltage_phase_three"), nil
}

// Currents returns per-phase current derived as I = P / V.
// Returns api.ErrNotAvailable if any phase voltage is zero (D-05) to avoid
// division by zero and to signal the runtime that this reading is unusable.
// util.Cached coalesces the two inner dataG() calls inside the TTL window.
func (c *Connection) Currents() (float64, float64, float64, error) {
	p1, p2, p3, err := c.PhasePowers()
	if err != nil {
		return 0, 0, 0, err
	}
	v1, v2, v3, err := c.PhaseVoltages()
	if err != nil {
		return 0, 0, 0, err
	}
	if v1 == 0 || v2 == 0 || v3 == 0 {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return p1 / v1, p2 / v2, p3 / v3, nil
}

// Compile-time assertions that the three methods satisfy the signature the
// generated decorator (Plan 02) expects: func() (float64, float64, float64, error).
// These catch signature drift at compile time rather than at go-generate time
// (Pitfall 5, 04-RESEARCH.md).
var (
	_ func() (float64, float64, float64, error) = (*Connection)(nil).PhasePowers
	_ func() (float64, float64, float64, error) = (*Connection)(nil).PhaseVoltages
	_ func() (float64, float64, float64, error) = (*Connection)(nil).Currents
)
