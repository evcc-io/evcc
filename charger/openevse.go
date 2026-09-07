package charger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RAR/go-openevse"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// OpenEVSE charger implementation. State arrives over the firmware's /ws
// websocket and control goes through its claims API; both live in
// github.com/RAR/go-openevse. Phase switching is opt-in (phases1p3p: true) for
// modified three-phase controllers that answer the $G7/$S7 RAPI commands.
type OpenEVSE struct {
	implement.Caps
	conn *openevse.Client
}

// openevseFaults names the controller states that evcc has no status F constant
// for, so Status() reports them as a named error instead.
var openevseFaults = map[int]string{
	5:  "diode check failed",
	6:  "gfci fault",
	7:  "no ground",
	8:  "stuck relay",
	9:  "gfci self-test failure",
	10: "over temperature",
	11: "over current",
}

func init() {
	registry.AddCtx("openevse", NewOpenEVSEFromConfig)
}

// NewOpenEVSEFromConfig creates an OpenEVSE charger from generic config
func NewOpenEVSEFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Cache    time.Duration // TODO deprecated, state is pushed by the firmware

		// Phases1p3p enables 1p/3p switching via the $S7 RAPI command of
		// modified 3-phase controllers; stock controllers do not support it
		Phases1p3p bool
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewOpenEVSE(ctx, cc.URI, cc.User, cc.Password, cc.Phases1p3p)
}

// NewOpenEVSE creates OpenEVSE charger
func NewOpenEVSE(ctx context.Context, uri, user, password string, phases1p3p bool) (api.Charger, error) {
	log := util.NewLogger("openevse").Redact(user, password, transport.BasicAuthHeader(user, password))

	client := request.NewClient(log)
	if user != "" && password != "" {
		client.Transport = transport.BasicAuth(user, password, client.Transport)
	}

	conn, err := openevse.New(uri, openevse.WithHTTPClient(client), openevse.WithLogger(openevseLogger{log}))
	if err != nil {
		return nil, err
	}

	// waits for the first status frame, so a wrong host or password fails the config test
	if err := conn.Connect(ctx); err != nil {
		return nil, err
	}

	c := &OpenEVSE{
		Caps: implement.New(),
		conn: conn,
	}

	if phases1p3p {
		if err := conn.EnablePhaseSwitching(); err != nil {
			conn.Close()
			return nil, err
		}
		implement.Has(c, implement.PhaseSwitcher(c.phases1p3p))
	}

	// evcc does not cancel a device's context on shutdown, so release the
	// claim through the shutdown hooks instead
	shutdown.Register(conn.Close)

	return c, nil
}

// openevseLogger adapts util.Logger to openevse.Logger
type openevseLogger struct {
	*util.Logger
}

func (l openevseLogger) Tracef(format string, args ...any) { l.TRACE.Printf(format, args...) }
func (l openevseLogger) Debugf(format string, args ...any) { l.DEBUG.Printf(format, args...) }
func (l openevseLogger) Warnf(format string, args ...any)  { l.WARN.Printf(format, args...) }
func (l openevseLogger) Errorf(format string, args ...any) { l.ERROR.Printf(format, args...) }

// Status implements the api.Charger interface
func (c *OpenEVSE) Status() (api.ChargeStatus, error) {
	res, err := c.conn.Status()
	if err != nil {
		return api.StatusNone, err
	}

	return openevseStatus(res)
}

// openevseStatus maps the controller state to a charge status:
//
//	1 not connected                  -> A
//	2 connected, 254 sleeping,
//	255 disabled, 4 vent required    -> B if a vehicle is connected, else A (evcc has no status D)
//	3 charging                       -> C
//	5-11 controller faults           -> named error (evcc has no status F constant)
func openevseStatus(res openevse.Status) (api.ChargeStatus, error) {
	switch res.State {
	case 1:
		return api.StatusA, nil
	case 2, 4, 254, 255:
		if res.Vehicle == 1 {
			return api.StatusB, nil
		}
		return api.StatusA, nil
	case 3:
		return api.StatusC, nil
	default:
		if name, ok := openevseFaults[res.State]; ok {
			return api.StatusNone, fmt.Errorf("charger fault: %s", name)
		}
		return api.StatusNone, fmt.Errorf("invalid status: %d", res.State)
	}
}

// Enabled implements the api.Charger interface
func (c *OpenEVSE) Enabled() (bool, error) {
	res, err := c.conn.Status()
	return res.Status == openevse.Enabled, err
}

// Enable implements the api.Charger interface
func (c *OpenEVSE) Enable(enable bool) error {
	return c.conn.Enable(enable)
}

// MaxCurrent implements the api.Charger interface
func (c *OpenEVSE) MaxCurrent(current int64) error {
	return c.conn.SetCurrent(int(current))
}

// phases1p3p implements the api.PhaseSwitcher interface
func (c *OpenEVSE) phases1p3p(phases int) error {
	return c.conn.SetThreePhase(phases == 3)
}

var _ api.CurrentGetter = (*OpenEVSE)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface. `pilot` is the firmware's
// arbitration result across all claims, so a higher-priority claim (manual override,
// limit) shows through here by design.
func (c *OpenEVSE) GetMaxCurrent() (float64, error) {
	res, err := c.conn.Status()
	return res.Pilot, err
}

var _ api.Meter = (*OpenEVSE)(nil)

// CurrentPower implements the api.Meter interface
func (c *OpenEVSE) CurrentPower() (float64, error) {
	res, err := c.conn.Status()
	return res.Power, err
}

var _ api.MeterEnergy = (*OpenEVSE)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *OpenEVSE) TotalEnergy() (float64, error) {
	res, err := c.conn.Status()
	return res.TotalEnergy, err
}

var _ api.ChargeRater = (*OpenEVSE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *OpenEVSE) ChargedEnergy() (float64, error) {
	res, err := c.conn.Status()
	return res.SessionEnergy / 1e3, err
}

var _ api.ChargeTimer = (*OpenEVSE)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (c *OpenEVSE) ChargeDuration() (time.Duration, error) {
	res, err := c.conn.Status()
	return time.Duration(res.Elapsed) * time.Second, err
}

var _ api.Identifier = (*OpenEVSE)(nil)

// Identify implements the api.Identifier interface. It returns the RFID tag that
// authorised the current session, if any.
func (c *OpenEVSE) Identify() ([]string, error) {
	res, err := c.conn.Status()
	if err != nil {
		return nil, err
	}

	if tag := openevseTag(res.RfidAuth); tag != "" {
		return []string{tag}, nil
	}

	return nil, nil
}

// openevseTag cleans the firmware's rfid_auth value. The firmware builds its
// "no tag" value from a '\0' char, so a tag consisting only of NUL bytes and/or
// whitespace is treated as empty.
func openevseTag(tag string) string {
	return strings.Trim(tag, "\x00 \t\n\r")
}
