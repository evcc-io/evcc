package charger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openevse"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// OpenEVSE charger implementation.
//
// State is received over the firmware's /ws websocket: the full status document on
// connect, then partial documents with only the changed keys. Control goes through
// the firmware's claims API (POST/DELETE /claims/{client}) at the firmware's fixed
// API priority, so the user's manual override, RFID, OCPP and limits still win while
// evcc wins over the built-in solar divert and schedules.
type OpenEVSE struct {
	*request.Helper
	log   *util.Logger
	uri   string
	wsURI string

	mu        sync.RWMutex
	status    openevse.Status
	connected bool
	err       error // last dial/read error, reported by the constructor's readiness wait

	ready     chan struct{} // closed once the first frame has been merged
	readyOnce sync.Once

	cancel context.CancelFunc // stops run(), used when construction fails

	// keepalive and read deadline, snapshotted from the package vars at construction
	// so that the reader goroutine never races with a test shortening them
	pingInterval time.Duration
	readTimeout  time.Duration

	claimMu sync.Mutex
	claim   *openevse.Claim // last claim written, nil until the first write
	enabled bool
	current int

	// overrideWarned guards against repeating the manual-override warning on every
	// claim write; it is reset once a status frame shows the override cleared.
	// Guarded by mu, alongside the status it derives from.
	overrideWarned bool

	// warnOverride logs the manual-override warning; a field so tests can observe
	// it without capturing log output.
	warnOverride func()
}

const overrideWarning = "manual override active on charger: evcc's claim is outranked until it is cleared (tap Auto on the charger dashboard, or DELETE /override)"

var errOpenEVSENotConnected = errors.New("websocket not connected")

// vars, not consts, so tests can shorten them
var (
	// openevsePingInterval is the keepalive interval. The firmware answers {"ping":1}
	// with {"pong":1}, so a healthy connection delivers a frame at least this often.
	openevsePingInterval = 30 * time.Second

	// openevseReadTimeout bounds every read. Without it a half-open TCP connection
	// blocks the reader forever while the charger keeps reporting stale state.
	openevseReadTimeout = 2*openevsePingInterval + 15*time.Second

	// openevseReadyTimeout is how long the constructor waits for the first frame
	openevseReadyTimeout = request.Timeout
)

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
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewOpenEVSE(ctx, cc.URI, cc.User, cc.Password)
}

// NewOpenEVSE creates OpenEVSE charger
func NewOpenEVSE(ctx context.Context, uri, user, password string) (api.Charger, error) {
	basicAuth := transport.BasicAuthHeader(user, password)
	log := util.NewLogger("openevse").Redact(user, password, basicAuth)

	c := &OpenEVSE{
		Helper: request.NewHelper(log),
		log:    log,
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		ready:  make(chan struct{}),

		pingInterval: openevsePingInterval,
		readTimeout:  openevseReadTimeout,
	}

	c.warnOverride = func() { c.log.WARN.Println(overrideWarning) }

	if user != "" && password != "" {
		c.Client.Transport = transport.BasicAuth(user, password, c.Client.Transport)
	}

	wsURI, err := parseURI(c.uri)
	if err != nil {
		return nil, err
	}
	c.wsURI = wsURI

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.run(ctx)

	// wait for the first usable frame so that a wrong host, wrong password or a
	// firmware without the claims API fails construction instead of silently
	// never delivering state
	select {
	case <-c.ready:
	case <-ctx.Done():
		c.cancel()
		return nil, ctx.Err()
	case <-time.After(openevseReadyTimeout):
		c.cancel()
		if err := c.lastError(); err != nil {
			return nil, fmt.Errorf("websocket: %w: %w", api.ErrTimeout, err)
		}
		return nil, fmt.Errorf("websocket: %w", api.ErrTimeout)
	}

	// evcc does not cancel a device's context on shutdown - ctx lives for the device
	// lifetime and is only cancelled on failure - so run()'s deferred release() never
	// fires from a normal SIGINT/SIGTERM. Register with evcc's shutdown hooks instead.
	// release() is idempotent (guarded by claim == nil under claimMu), so it is safe
	// to also run via the deferred call in run() when ctx is cancelled (embedding, tests).
	shutdown.Register(c.release)

	return c, nil
}

// run keeps the websocket connected until ctx is cancelled
func (c *OpenEVSE) run(ctx context.Context) {
	defer c.release()

	bo := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(0),
		backoff.WithMaxInterval(30*time.Second),
	)

	for ctx.Err() == nil {
		c.log.DEBUG.Println("websocket: connecting")

		conn, _, err := websocket.Dial(ctx, c.wsURI, &websocket.DialOptions{HTTPClient: c.Client})
		if err != nil {
			c.setError(err)

			if ctx.Err() == nil {
				c.log.ERROR.Printf("websocket: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(bo.NextBackOff()):
			}

			continue
		}

		if err := c.handleConnection(ctx, conn, bo); err != nil {
			c.setError(err)

			if ctx.Err() == nil {
				c.log.ERROR.Printf("websocket: %v", err)
			}
		}

		c.setConnected(false)
	}
}

// handleConnection reads frames until the connection fails. The first frame of a
// connection is the full status; after merging it successfully the charger is
// readable and the reconnect backoff is reset, proving the connection is usable
// (a device that accepts the websocket and then immediately drops it must not
// reset the backoff and cause a hot reconnect loop).
func (c *OpenEVSE) handleConnection(ctx context.Context, conn *websocket.Conn, bo *backoff.ExponentialBackOff) error {
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// keepalive: the firmware answers {"ping":1} with {"pong":1}
	go func() {
		ticker := time.NewTicker(c.pingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.Write(ctx, websocket.MessageText, []byte(`{"ping":1}`)); err != nil {
					c.log.ERROR.Printf("websocket: ping: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	first := true
	for {
		// bound every read: the firmware answers each ping, so a healthy connection
		// always delivers a frame well inside this window. A half-open connection
		// times out here instead of blocking forever with stale state marked valid.
		readCtx, readCancel := context.WithTimeout(ctx, c.readTimeout)
		typ, b, err := conn.Read(readCtx)
		readCancel()

		if err != nil {
			return err
		}
		if typ != websocket.MessageText {
			continue
		}

		c.log.TRACE.Printf("websocket: %s", b)

		err = c.merge(b)
		if err != nil {
			c.log.ERROR.Printf("websocket: bad frame: %v", err)
		}

		// a partial merge still delivers real state for the fields that did decode
		// (the firmware may add keys with unexpected types); anything else - a
		// syntax error, non-object frame, etc - is not usable state
		var typeErr *json.UnmarshalTypeError
		ok := err == nil || errors.As(err, &typeErr)

		if first && ok {
			first = false
			bo.Reset()
			c.setConnected(true)

			// the first connection of the charger's lifetime releases the constructor;
			// no claim can have been written before it returns, so only a reconnect
			// (a firmware reboot drops all claims) has anything to re-assert
			var initial bool
			c.readyOnce.Do(func() {
				initial = true
				close(c.ready)
			})

			if !initial {
				go func() {
					if err := c.reassertClaim(); err != nil {
						c.log.WARN.Printf("reassert claim: %v", err)
					}
				}()
			}
		}
	}
}

// merge applies a full or partial status document; keys absent from the frame keep their value
func (c *OpenEVSE) merge(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := json.Unmarshal(b, &c.status)

	// the override warning is re-armed only once the charger has confirmed the
	// override is actually cleared, so a flapping override doesn't get lost in
	// a burst of unwarned claim writes
	if c.status.ManualOverride == 0 {
		c.overrideWarned = false
	}

	return err
}

func (c *OpenEVSE) setConnected(connected bool) {
	c.mu.Lock()
	c.connected = connected
	c.mu.Unlock()
}

func (c *OpenEVSE) setError(err error) {
	c.mu.Lock()
	c.err = err
	c.mu.Unlock()
}

// lastError returns the most recent dial or read error, nil if there was none yet
func (c *OpenEVSE) lastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

// get returns the current status or an error while the websocket is down
func (c *OpenEVSE) get() (openevse.Status, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return openevse.Status{}, errOpenEVSENotConnected
	}

	return c.status, nil
}

// Status implements the api.Charger interface
func (c *OpenEVSE) Status() (api.ChargeStatus, error) {
	res, err := c.get()
	if err != nil {
		return api.StatusNone, err
	}

	/*
		0: "unknown",
		1: "not connected",
		2: "connected",
		3: "charging",
		4: "vent required",
		5: "diode check failed",
		6: "gfci fault",
		7: "no ground",
		8: "stuck relay",
		9: "gfci self-test failure",
		10: "over temperature",
		11: "over current",
		254: "sleeping",
		255: "disabled"
	*/

	switch res.State {
	case 1:
		return api.StatusA, nil
	case 2, 254, 255:
		if res.Vehicle == 1 {
			return api.StatusB, nil
		}
		return api.StatusA, nil
	case 3:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", res.State)
	}
}

// Enabled implements the api.Charger interface
func (c *OpenEVSE) Enabled() (bool, error) {
	res, err := c.get()
	return res.Status == openevse.Enabled, err
}

// Enable implements the api.Charger interface
func (c *OpenEVSE) Enable(enable bool) error {
	c.claimMu.Lock()
	defer c.claimMu.Unlock()

	c.enabled = enable
	return c.setClaim()
}

// MaxCurrent implements the api.Charger interface
func (c *OpenEVSE) MaxCurrent(current int64) error {
	c.claimMu.Lock()
	defer c.claimMu.Unlock()

	c.current = int(current)
	return c.setClaim()
}

func (c *OpenEVSE) claimURI() string {
	return fmt.Sprintf("%s/claims/%d", c.uri, openevse.ClientID)
}

// setClaim posts the full claim built from the desired state; caller holds claimMu
func (c *OpenEVSE) setClaim() error {
	c.warnIfOverridden()

	claim := openevse.Claim{
		State:         openevse.Disabled,
		ChargeCurrent: c.current,
	}
	if c.enabled {
		claim.State = openevse.Enabled
	}

	if err := c.postClaim(claim); err != nil {
		return err
	}

	c.claim = &claim
	return nil
}

// warnIfOverridden logs once that the firmware's manual override outranks evcc's
// claim, whenever the last received status shows it active. It logs again only
// after a status frame has shown the override inactive in between.
func (c *OpenEVSE) warnIfOverridden() {
	c.mu.Lock()
	warn := c.status.ManualOverride == 1 && !c.overrideWarned
	if warn {
		c.overrideWarned = true
	}
	c.mu.Unlock()

	if warn {
		c.warnOverride()
	}
}

func (c *OpenEVSE) postClaim(claim openevse.Claim) error {
	req, err := request.New(http.MethodPost, c.claimURI(), request.MarshalJSON(claim), request.JSONEncoding)
	if err != nil {
		return err
	}

	_, err = c.DoBody(req)
	return err
}

// reassertClaim re-posts the last claim after a (re)connect: a firmware reboot drops all claims
func (c *OpenEVSE) reassertClaim() error {
	c.claimMu.Lock()
	defer c.claimMu.Unlock()

	if c.claim == nil {
		return nil
	}

	return c.postClaim(*c.claim)
}

// release deletes evcc's claim on shutdown so the EVSE falls back to its own defaults
func (c *OpenEVSE) release() {
	c.claimMu.Lock()
	defer c.claimMu.Unlock()

	if c.claim == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := request.New(http.MethodDelete, c.claimURI(), nil)
	if err == nil {
		_, err = c.DoBody(req.WithContext(ctx))
	}

	if err != nil {
		c.log.WARN.Printf("release claim: %v", err)
	}

	c.claim = nil
}

var _ api.CurrentGetter = (*OpenEVSE)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface. `pilot` is the firmware's
// arbitration result across all claims, so a higher-priority claim (manual override,
// limit) shows through here by design.
func (c *OpenEVSE) GetMaxCurrent() (float64, error) {
	res, err := c.get()
	return res.Pilot, err
}

var _ api.Meter = (*OpenEVSE)(nil)

// CurrentPower implements the api.Meter interface
func (c *OpenEVSE) CurrentPower() (float64, error) {
	res, err := c.get()
	return res.Power, err
}

var _ api.MeterEnergy = (*OpenEVSE)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *OpenEVSE) TotalEnergy() (float64, error) {
	res, err := c.get()
	return res.TotalEnergy, err
}

var _ api.ChargeRater = (*OpenEVSE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *OpenEVSE) ChargedEnergy() (float64, error) {
	res, err := c.get()
	return res.SessionEnergy / 1e3, err
}

var _ api.ChargeTimer = (*OpenEVSE)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (c *OpenEVSE) ChargeDuration() (time.Duration, error) {
	res, err := c.get()
	return time.Duration(res.Elapsed) * time.Second, err
}
