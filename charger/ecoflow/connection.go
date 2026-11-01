package ecoflow

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

// Transport is the MQTT client a Connection runs on. The real one talks to the
// EcoFlow app broker; tests use ecoflowtest.Transport.
type Transport interface {
	Subscribe(topic string, qos byte, handler func(payload []byte)) error
	Publish(ctx context.Context, topic string, qos byte, payload []byte) error
}

// sharedConn is a connection in use by refs chargers of one account.
type sharedConn struct {
	conn   *Connection
	secret [32]byte
	refs   int
}

var (
	mu          sync.Mutex
	connections = make(map[string]*sharedConn)
	redacted    = make(map[[32]byte]bool) // static secrets already on the shared logger
)

// Dialer connects to the broker with the session's credentials. onLost is
// called once when the established connection drops.
type Dialer func(session *Session, onLost func(error)) (Transport, error)

// Connection is one app MQTT session of an EcoFlow account. All devices of the
// account share it, because EcoFlow drops concurrent sessions of one account.
type Connection struct {
	log       *util.Logger
	transport *switchTransport
	userID    string
	seq       atomic.Uint64

	mu        sync.Mutex
	wallboxes []*Wallbox
	traffic   time.Time // last payload from any device
	probed    time.Time // last get-all of the watchdog

	// reconnect, nil for a connection on a fixed transport
	login        func() (*Session, error)
	dial         Dialer
	clock        clock.Clock
	sleep        func(time.Duration)
	session      *Session
	loggedIn     time.Time
	reconnecting atomic.Bool
	revived      time.Time // last reconnect of the watchdog
	strikes      int       // watchdog reconnects without traffic since

	ctx    context.Context // cancelled by Close, bounds every publish
	cancel context.CancelFunc
}

// NewSharedConnection returns the connection of the account, logging in and
// connecting on first use. It closes once the contexts of all its users are done.
func NewSharedConnection(ctx context.Context, user, password string, app AppKey) (*Connection, error) {
	return sharedConnection(ctx, user, password, func() (*Connection, error) {
		// the logger is shared: static secrets are added once, the session ones replace their predecessors
		log := util.NewLogger("ecoflow")
		if key := sha256.Sum256([]byte(user + "\x00" + password + "\x00" + app.Secret)); !redacted[key] {
			log.Redact(user, password, base64.StdEncoding.EncodeToString([]byte(password)), app.Secret)
			redacted[key] = true
		}
		redactSession := log.RotatingSlot()

		login := func() (*Session, error) {
			return Login(log, redactSession, user, password)
		}
		dial := func(session *Session, onLost func(error)) (Transport, error) {
			return dial(log, session, app, onLost)
		}

		conn, err := Connect(log, login, dial, clock.New(), nil)
		if err == nil {
			go conn.watch()
		}
		return conn, err
	})
}

// sharedConnection keeps one connection per account. A changed password
// replaces it, because EcoFlow drops concurrent sessions of one account anyway.
func sharedConnection(ctx context.Context, user, password string, connect func() (*Connection, error)) (*Connection, error) {
	mu.Lock()
	defer mu.Unlock()

	secret := sha256.Sum256([]byte(password))

	e, ok := connections[user]
	if ok && e.secret != secret {
		e.conn.Close()
		delete(connections, user)
		ok = false
	}

	if !ok {
		conn, err := connect()
		if err != nil {
			return nil, err
		}
		e = &sharedConn{conn: conn, secret: secret}
		connections[user] = e
	}

	e.refs++
	go func() {
		<-ctx.Done()

		mu.Lock()
		defer mu.Unlock()
		if e.refs--; e.refs == 0 {
			e.conn.Close()
			if connections[user] == e {
				delete(connections, user)
			}
		}
	}()

	return e.conn, nil
}

// NewConnection creates a connection on an established transport. It does not
// reconnect.
func NewConnection(log *util.Logger, transport Transport, userID string) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		log:       log,
		transport: newSwitchTransport(transport),
		userID:    userID,
		clock:     clock.New(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Connect logs in, connects and reconnects with backoff after a loss.
// sleep waits out the backoff; nil waits on the clock until Close.
func Connect(log *util.Logger, login func() (*Session, error), dial Dialer, clk clock.Clock, sleep func(time.Duration)) (*Connection, error) {
	session, err := login()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Connection{
		log:       log,
		transport: newSwitchTransport(nil),
		userID:    session.UserID,
		login:     login,
		dial:      dial,
		clock:     clk,
		sleep:     sleep,
		session:   session,
		loggedIn:  clk.Now(),
		traffic:   clk.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}

	if err := c.connect(); err != nil {
		cancel()
		return nil, err
	}

	return c, nil
}

// connect dials with the current session and restores subscriptions and
// snapshots on the new transport.
func (c *Connection) connect() error {
	gen := c.transport.generation() + 1
	onLost := func(err error) {
		if c.transport.generation() != gen {
			return // a connection that was already replaced
		}
		c.log.WARN.Printf("connection lost: %v", err)
		go c.reconnect()
	}

	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	t, err := c.dial(session, onLost)
	if err != nil {
		return err
	}

	if err := c.transport.attach(t); err != nil {
		c.transport.detach()
		return fmt.Errorf("subscribe: %w", err)
	}

	c.mu.Lock()
	wallboxes := append([]*Wallbox(nil), c.wallboxes...)
	c.mu.Unlock()

	for _, wb := range wallboxes {
		if err := wb.requestAll(); err != nil {
			c.log.ERROR.Printf("%s: get-all: %v", wb.serial, err)
		}
	}

	return nil
}

func (c *Connection) setSession(session *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.session = session
	c.loggedIn = c.clock.Now()
}

// Watchdog timing.
const (
	checkInterval = time.Minute
	loginRenewal  = 20 * time.Hour
	probeAfter    = 90 * time.Second   // heartbeat gap after which a wallbox gets a get-all
	staleAfter    = 1200 * time.Second // heartbeats normally arrive every 60 s
	snapshotGrace = 5 * time.Minute    // wait for an answer to the get-all
	maxRevival    = 6 * time.Hour      // longest wait between watchdog reconnects
)

// publishTimeout bounds the wait for the broker's PUBACK.
const publishTimeout = 10 * time.Second

// Close stops reconnecting and watching and drops the broker connection.
func (c *Connection) Close() {
	c.cancel()
	c.transport.close()
}

func (c *Connection) isClosed() bool {
	return c.ctx.Err() != nil
}

// publish sends a payload, waiting at most publishTimeout for the broker.
func (c *Connection) publish(topic string, payload []byte) error {
	ctx, cancel := context.WithTimeout(c.ctx, publishTimeout)
	defer cancel()
	return c.transport.Publish(ctx, topic, 1, payload)
}

// received records traffic on the connection.
func (c *Connection) received() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.traffic = c.clock.Now()
}

// watch runs check periodically until Close.
func (c *Connection) watch() {
	ticker := c.clock.Ticker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.check()
		}
	}
}

// check renews the login for the next connect and revives silent telemetry:
// a get-all to every device whose heartbeat is missing for probeAfter, a new
// connection only when no device answers within snapshotGrace after staleAfter
// of silence. A device that is offline must not reconnect the account
// every few minutes, so watchdog reconnects back off while nothing arrives.
func (c *Connection) check() {
	if c.login == nil || c.isClosed() {
		return
	}

	c.mu.Lock()
	due := c.clock.Since(c.loggedIn) >= loginRenewal
	wallboxes := append([]*Wallbox(nil), c.wallboxes...)
	c.mu.Unlock()

	if due {
		session, err := c.login()
		if err != nil {
			c.log.ERROR.Printf("login renewal: %v", err)
		} else {
			c.log.DEBUG.Println("login renewed")
			c.setSession(session)
		}
	}

	if c.reconnecting.Load() {
		return
	}

	for _, wb := range wallboxes {
		if wb.probe() {
			c.mu.Lock()
			c.probed = c.clock.Now()
			c.mu.Unlock()
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.traffic.After(c.revived) {
		c.strikes = 0
	}

	silent := c.clock.Since(c.traffic) >= staleAfter
	unanswered := c.probed.After(c.traffic) && c.clock.Since(c.probed) >= snapshotGrace
	if !silent || !unanswered || c.clock.Since(c.revived) < revival(c.strikes) {
		return
	}

	c.log.WARN.Printf("no telemetry since %v, reconnecting", c.clock.Since(c.traffic).Round(time.Second))
	c.revived = c.clock.Now()
	c.strikes++
	go c.reconnect()
}

// revival is the wait after n watchdog reconnects without traffic: none at
// first, then staleAfter·2ⁿ, at most maxRevival.
func revival(n int) time.Duration {
	if n == 0 {
		return 0
	}
	return min(staleAfter<<n, maxRevival)
}

// backoff is the wait before reconnect attempt n: 5·2ⁿ s, at most 60 s.
func backoff(n int) time.Duration {
	if n >= 4 {
		return time.Minute
	}
	return 5 * time.Second << n
}

// reconnect replaces the transport, retrying with backoff until it succeeds
// or the connection is closed. Only one reconnect runs at a time.
func (c *Connection) reconnect() {
	if c.dial == nil || c.isClosed() || !c.reconnecting.CompareAndSwap(false, true) {
		return
	}
	defer c.reconnecting.Store(false)

	c.transport.detach()

	var relogin bool
	for n := 0; ; n++ {
		c.log.DEBUG.Printf("reconnecting in %v (attempt %d)", backoff(n), n+1)
		if !c.pause(backoff(n)) {
			return
		}

		if relogin {
			c.log.DEBUG.Println("logging in again before reconnect")
			session, err := c.login()
			if err != nil {
				c.log.ERROR.Printf("login: %v", err)
				continue
			}
			c.setSession(session)
		}

		err := c.connect()
		if err == nil {
			c.log.DEBUG.Println("reconnected")
			return
		}

		c.log.WARN.Printf("reconnect: %v", err)
		relogin = needsLogin(err)
	}
}

// pause waits d and reports whether the connection is still open.
func (c *Connection) pause(d time.Duration) bool {
	if c.sleep != nil {
		c.sleep(d)
	} else {
		select {
		case <-c.clock.After(d):
		case <-c.ctx.Done():
		}
	}
	return !c.isClosed()
}

// WithClock sets the clock that ages the telemetry of the connection's devices.
// Set it before creating devices.
func (c *Connection) WithClock(clk clock.Clock) *Connection {
	c.clock = clk
	return c
}

// nextSeq returns the next frame sequence number of this connection.
func (c *Connection) nextSeq() uint64 {
	return c.seq.Add(1)
}

// Wallbox is a PowerPulse reached through a Connection.
type Wallbox struct {
	log       *util.Logger
	conn      *Connection
	serial    string
	gateway   string // PowerOcean that writes go through
	timeout   time.Duration
	heartbeat *util.Monitor[Heartbeat]

	cmd sync.Mutex // serializes start, stop and phase writes

	mu         sync.Mutex
	seen       time.Time // last heartbeat, or creation
	probed     time.Time // last get-all of the watchdog
	settings   Settings  // last 241/44
	settingsAt time.Time

	// last commands, telemetry may lag behind them
	release command[bool]
	current command[uint64] // session setpoint, 0.1 A
	phases  command[uint64]

	switching *phaseSwitch // phase switch under way while charging
}

// Wallbox subscribes to the telemetry of the PowerPulse with the given serial
// and requests a snapshot. Telemetry older than timeout counts as outdated.
func (c *Connection) Wallbox(serial, gateway string, timeout time.Duration) (*Wallbox, error) {
	wb := &Wallbox{
		log:       c.log,
		conn:      c,
		serial:    serial,
		gateway:   gateway,
		timeout:   timeout,
		heartbeat: util.NewMonitor[Heartbeat](timeout).WithClock(c.clock),
		seen:      c.clock.Now(),
	}

	if err := c.transport.Subscribe("/app/device/property/"+serial, 0, wb.handle); err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	c.log.DEBUG.Printf("%s: subscribed, writes via gateway %s", serial, gateway)

	if err := wb.requestAll(); err != nil {
		return nil, fmt.Errorf("get-all: %w", err)
	}

	c.mu.Lock()
	c.wallboxes = slices.DeleteFunc(c.wallboxes, func(x *Wallbox) bool { return x.serial == serial })
	c.wallboxes = append(c.wallboxes, wb)
	c.mu.Unlock()

	return wb, nil
}

// probe sends a get-all when the wallbox has been silent for probeAfter, once
// per silence and again every staleAfter while it lasts. A get-all brings back
// heartbeats that stopped while other frames kept arriving. It reports whether
// it sent one.
func (wb *Wallbox) probe() bool {
	c := wb.conn

	wb.mu.Lock()
	seen, probed := wb.seen, wb.probed
	due := c.clock.Since(seen) >= probeAfter && (probed.Before(seen) || c.clock.Since(probed) >= staleAfter)
	if due {
		wb.probed = c.clock.Now()
	}
	wb.mu.Unlock()

	if !due {
		return false
	}

	c.log.DEBUG.Printf("%s: no heartbeat since %v, requesting snapshot", wb.serial, c.clock.Since(seen).Round(time.Second))
	if err := wb.requestAll(); err != nil {
		c.log.ERROR.Printf("%s: get-all: %v", wb.serial, err)
	}
	return true
}

// requestAll asks the wallbox for a full snapshot.
func (wb *Wallbox) requestAll() error {
	c := wb.conn

	seq := c.nextSeq()
	wb.log.TRACE.Printf("%s: get-all seq=%d", wb.serial, seq)
	topic := fmt.Sprintf("/app/%s/%s/thing/property/get", c.userID, wb.serial)
	return c.publish(topic, BuildGetAll(seq))
}

// handle decodes one telemetry payload. Payloads that are not heartbeats are
// ignored.
func (wb *Wallbox) handle(payload []byte) {
	wb.conn.received()
	wb.log.TRACE.Printf("%s: recv %x", wb.serial, payload)
	headers, err := ParseMessage(payload)
	if err != nil {
		wb.log.TRACE.Printf("%s: undecodable payload: %v", wb.serial, err)
		return
	}

	for _, h := range headers {
		fn, id, ok := h.Command()
		switch {
		case ok && fn == 2 && id == 33:
			for _, pdata := range h.Pdata() {
				wb.handleHeartbeat(pdata)
			}
		case ok && fn == 241 && id == 44:
			for _, pdata := range h.Pdata() {
				wb.handleSettings(pdata)
			}
		}
	}
}

// handleHeartbeat stores a 2/33 heartbeat.
func (wb *Wallbox) handleHeartbeat(pdata []byte) {
	hb, err := DecodeHeartbeat(pdata)
	if err != nil {
		wb.log.DEBUG.Printf("%s: heartbeat: %v", wb.serial, err)
		return
	}
	now := wb.conn.clock.Now()
	wb.mu.Lock()
	wb.seen = now
	wb.mu.Unlock()

	wb.heartbeat.SetFunc(func(prev Heartbeat) Heartbeat {
		// some heartbeats carry no energy counter, keep the last one
		if hb.TotalEnergyWh == 0 {
			hb.TotalEnergyWh = prev.TotalEnergyWh
		}
		return hb
	})
	wb.log.TRACE.Printf("%s: heartbeat %s", wb.serial, hb)
	wb.verifyPhases(Update{Heartbeat: &hb}, now)
	wb.verifyCurrent(Update{Heartbeat: &hb}, now)
	wb.verifyRelease(hb, now)
	wb.stopped(hb.State)
}

// handleSettings records a 241/44 report. It
// only flows while the EcoFlow app is active, so it is no sign of life for the watchdog.
func (wb *Wallbox) handleSettings(pdata []byte) {
	s, err := DecodeSettings(pdata)
	if err != nil {
		// the shorter 241/44 variant without settings is routine
		if !errors.Is(err, ErrNoSettings) {
			wb.log.TRACE.Printf("%s: settings: %v", wb.serial, err)
		}
		return
	}
	now := wb.conn.clock.Now()
	wb.mu.Lock()
	changed := s != wb.settings || wb.settingsAt.IsZero()
	wb.settings, wb.settingsAt = s, now
	wb.mu.Unlock()

	wb.verifyPhases(Update{Settings: &s}, now)
	wb.verifyCurrent(Update{Settings: &s}, now)

	// 241/44 arrives at about 1 Hz, log changes only
	if changed {
		wb.log.TRACE.Printf("%s: settings %+v", wb.serial, s)
	}
}

// latestSettings returns the last 241/44 report and when it arrived unless it
// is older than the timeout.
func (wb *Wallbox) latestSettings() (Settings, time.Time, bool) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	if wb.settingsAt.IsZero() || wb.conn.clock.Since(wb.settingsAt) > wb.timeout {
		return Settings{}, time.Time{}, false
	}
	return wb.settings, wb.settingsAt, true
}

// latest returns the latest fresh heartbeat without waiting for a first one.
func (wb *Wallbox) latest() (Heartbeat, bool) {
	select {
	case <-wb.heartbeat.Done():
	default:
		return Heartbeat{}, false
	}
	hb, err := wb.heartbeat.Get()
	return hb, err == nil
}

// Heartbeat returns the latest heartbeat, or api.ErrOutdated when none
// arrived within the timeout.
func (wb *Wallbox) Heartbeat() (Heartbeat, error) {
	return wb.heartbeat.Get()
}

// ConnackError is a connect the broker refused with a CONNACK return code.
type ConnackError struct {
	Code byte
}

func (e *ConnackError) Error() string {
	return fmt.Sprintf("connection refused: CONNACK %d", e.Code)
}

// needsLogin reports whether the broker rejected the MQTT credentials
// (MQTT 3.1.1 codes 4 and 5, MQTT 5 codes 134 and 135).
func needsLogin(err error) bool {
	var ce *ConnackError
	if !errors.As(err, &ce) {
		return false
	}
	switch ce.Code {
	case 4, 5, 134, 135:
		return true
	}
	return false
}
