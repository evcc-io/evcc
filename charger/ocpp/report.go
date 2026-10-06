package ocpp

// One-way OCPP reporting client, one connection per loadpoint. Unlike the
// forwarder (which relays real OCPP frames between a charger that already
// speaks OCPP to evcc and an upstream Central System), this dials OUT to an
// upstream Central System and SYNTHESIZES OCPP messages from evcc's own
// charging session lifecycle - so it works for any charger type (Modbus
// etc.), not just OCPP-native ones. It never accepts remote control from the
// upstream: the upstream here is a billing/certification backend for a
// charger evcc already fully controls itself, not an access-control
// authority for it (unlike the forwarder's upstream, which can legitimately
// be one) - RemoteStartTransaction/RemoteStopTransaction and UnlockConnector
// are honestly rejected rather than translated into loadpoint control.
//
// Design: evcc-io/evcc#32989 ("no need to support locking/unlocking the
// charger, just logging the charge sessions would be sufficient").

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

var reportLog = util.NewLogger("ocpp-report")

// defaultMeterInterval throttles intermediate MeterValues sent while a
// session is active - core calls ReportMeterValue every loadpoint tick
// (whenever charged energy increases), but OCPP backends expect samples on
// the order of a typical MeterValueSampleInterval, not the loadpoint's own
// (often much shorter) update cadence. Mirrors the forwarder's per-charger
// meterInterval throttle (forwarder.go), fixed rather than absorbed from
// upstream since report has no ChangeConfiguration channel to absorb it from.
const (
	defaultMeterInterval     = 60 * time.Second
	defaultHeartbeatInterval = 5 * time.Minute
)

// ReportRule configures reporting a loadpoint's charging sessions to an
// upstream OCPP 1.6J Central System. One-way: evcc reports, it never accepts
// remote control from the upstream (evcc-io/evcc#32989).
type ReportRule struct {
	// LoadpointId is the loadpoint index the rule is bound to; LoadpointTitle is
	// display only and may change, so it never keys anything
	LoadpointId    int    `json:"loadpointId" yaml:"loadpointId"`
	LoadpointTitle string `json:"loadpointTitle" yaml:"loadpointTitle"`
	UpstreamURL    string `json:"upstreamUrl" yaml:"upstreamUrl"`
	// StationID is mandatory - no more "evcc-<loadpoint>" fallback; a rule
	// must name its own station explicitly.
	StationID string `json:"stationId" yaml:"stationId"`
	// IdTag is mandatory: the underlying ocpp-go library validates it
	// `required` on every Authorize/StartTransaction, so an empty value
	// would silently and permanently fail every session (see idTag's
	// removed fallback - it used to default to "EVCC").
	IdTag    string `json:"idTag" yaml:"idTag"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Insecure bool   `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	CaCert   string `json:"caCert,omitempty" yaml:"caCert,omitempty"`
}

func (r ReportRule) Redacted() ReportRule {
	r.Password = util.Masked(r.Password)
	r.CaCert = util.Masked(r.CaCert)
	return r
}

// sameConnection reports whether two rules dial upstream identically (no
// reconnect needed).
func (r ReportRule) sameConnection(o ReportRule) bool {
	return r.UpstreamURL == o.UpstreamURL && r.StationID == o.StationID &&
		r.Username == o.Username && r.Password == o.Password &&
		r.Insecure == o.Insecure && r.CaCert == o.CaCert
}

// ReportSessionStatus is a snapshot of one report connection for the UI.
type ReportSessionStatus struct {
	LoadpointId       int    `json:"loadpointId"`
	LoadpointTitle    string `json:"loadpointTitle"`
	UpstreamURL       string `json:"upstreamUrl"`
	UpstreamConnected bool   `json:"upstreamConnected"`
	IdTagStatus       string `json:"idTagStatus,omitempty"`
	Error             string `json:"error,omitempty"`
}

var (
	reportMu sync.RWMutex
	// reportEnabled is the master switch, independent of the configured
	// rules below - disabling it tears down every connection without
	// discarding the rules, so re-enabling reconnects them as-is. Defaults
	// true so a fresh boot with no explicit setting behaves like before this
	// switch existed.
	reportEnabled = true
	reportRules   []ReportRule
	connections   = make(map[int]*reportConnection) // keyed by LoadpointId
	reportErrors  = make(map[int]string)

	reportCbMu      sync.Mutex
	reportUpdatedCb func()
)

// SetReportUpdated registers a callback fired when a connection's status changes.
func SetReportUpdated(cb func()) {
	reportCbMu.Lock()
	reportUpdatedCb = cb
	reportCbMu.Unlock()
}

func notifyReportUpdated() {
	reportCbMu.Lock()
	cb := reportUpdatedCb
	reportCbMu.Unlock()
	if cb != nil {
		cb()
	}
}

// ReportEnabled returns true when at least one report rule is configured.
func ReportEnabled() bool {
	reportMu.RLock()
	defer reportMu.RUnlock()
	return len(reportRules) > 0
}

// ReportGloballyEnabled returns the master switch's current state.
func ReportGloballyEnabled() bool {
	reportMu.RLock()
	defer reportMu.RUnlock()
	return reportEnabled
}

// SetReportEnabled toggles the master switch. Disabling closes every
// connection without touching the configured rules; re-enabling reconnects
// them exactly as configured, without needing to resave anything.
func SetReportEnabled(enabled bool) {
	reportMu.Lock()
	if reportEnabled == enabled {
		reportMu.Unlock()
		return
	}
	reportEnabled = enabled
	rules := slices.Clone(reportRules)
	reportMu.Unlock()

	ApplyReportRules(rules)
}

// ReportRules returns the currently configured report rules.
func ReportRules() []ReportRule {
	reportMu.RLock()
	defer reportMu.RUnlock()
	return slices.Clone(reportRules)
}

// GetReportStatus returns a snapshot of all configured report connections.
func GetReportStatus() []ReportSessionStatus {
	reportMu.RLock()
	defer reportMu.RUnlock()

	out := make([]ReportSessionStatus, 0, len(reportRules))
	for _, r := range reportRules {
		st := ReportSessionStatus{
			LoadpointId:    r.LoadpointId,
			LoadpointTitle: r.LoadpointTitle,
			UpstreamURL:    strings.TrimRight(r.UpstreamURL, "/"),
		}
		if conn, ok := connections[r.LoadpointId]; ok && conn.cp.IsConnected() {
			st.UpstreamConnected = true
			conn.mu.Lock()
			st.IdTagStatus = conn.idTagStatus
			conn.mu.Unlock()
		} else if msg, ok := reportErrors[r.LoadpointId]; ok {
			st.Error = msg
		}
		out = append(out, st)
	}
	return out
}

func recordReportError(id int, msg string) {
	reportMu.Lock()
	reportErrors[id] = msg
	reportMu.Unlock()
	notifyReportUpdated()
}

func clearReportError(id int) {
	reportMu.Lock()
	_, had := reportErrors[id]
	delete(reportErrors, id)
	reportMu.Unlock()
	if had {
		notifyReportUpdated()
	}
}

// ApplyReportRules replaces the report rules and (re)dials connections.
func ApplyReportRules(rules []ReportRule) {
	reportMu.Lock()
	reportRules = rules

	// while globally disabled, valid stays empty: every existing connection
	// below is torn down as stale and none get (re)started, but reportRules
	// still holds the real config for status/config endpoints and for
	// SetReportEnabled to reconnect from once re-enabled
	valid := make(map[int]bool, len(rules))
	if reportEnabled {
		for _, r := range rules {
			valid[r.LoadpointId] = true
		}
	}

	var stale []*reportConnection
	for id, conn := range connections {
		if !valid[id] {
			stale = append(stale, conn)
			delete(connections, id)
			delete(reportErrors, id)
		}
	}

	var toStart []*reportConnection
	if reportEnabled {
		for _, r := range rules {
			if old, ok := connections[r.LoadpointId]; ok {
				if old.rule.sameConnection(r) {
					old.rule = r // idTag etc. may have changed, doesn't need a reconnect
					continue
				}
				stale = append(stale, old)
			}
			conn := newReportConnection(r)
			connections[r.LoadpointId] = conn
			toStart = append(toStart, conn)
		}
	}
	reportMu.Unlock()

	for _, conn := range stale {
		conn.close()
	}
	for _, conn := range toStart {
		go conn.run()
	}

	notifyReportUpdated()
}

// reportConnection is one loadpoint's OCPP client connection to its upstream.
type reportConnection struct {
	title string
	rule  ReportRule
	cp    ocpp16.ChargePoint
	jobs  chan func()
	done  chan struct{}

	mu     sync.Mutex
	booted bool

	// events not yet confirmed upstream, in the order they happened; reconcile
	// drains the head and stops at the first failure, so a failed message is
	// retried by the next call and events queued while offline keep their times
	events        []reportEvent
	sessionActive bool
	transactionId *int
	lastWh        float64   // last register value of the session, re-sent on request
	lastAt        time.Time // when lastWh was taken
	idTagStatus   string    // last idTag status the upstream answered with

	// intermediate MeterValues throttle - reconcile always runs on this
	// connection's single worker goroutine (run's job loop), so these two are
	// only ever touched from there and need no mutex, unlike the fields above
	meterInterval time.Duration
	lastMeterSent time.Time
}

func newReportConnection(rule ReportRule) *reportConnection {
	conn := &reportConnection{
		title:         rule.LoadpointTitle,
		rule:          rule,
		jobs:          make(chan func(), 16),
		done:          make(chan struct{}),
		meterInterval: defaultMeterInterval,
	}

	var opts []ws.ClientOpt
	if rule.Insecure || rule.CaCert != "" {
		tlsConfig := &tls.Config{InsecureSkipVerify: rule.Insecure}
		if rule.CaCert != "" {
			pool := x509.NewCertPool()
			if pool.AppendCertsFromPEM([]byte(rule.CaCert)) {
				tlsConfig.RootCAs = pool
			}
		}
		opts = append(opts, ws.WithClientTLSConfig(tlsConfig))
	}

	client := ws.NewClient(opts...)
	client.SetRequestedSubProtocol(types.V16Subprotocol)
	if rule.Username != "" || rule.Password != "" {
		client.SetBasicAuth(rule.Username, rule.Password)
	}

	stationID := rule.StationID

	endpoint := ocppj.NewClient(stationID, client, nil, nil, core.Profile, remotetrigger.Profile)
	handler := &reportHandler{conn: conn}

	cp := ocpp16.NewChargePoint(stationID, endpoint, client)
	cp.SetCoreHandler(handler)
	cp.SetRemoteTriggerHandler(handler)
	conn.cp = cp

	endpoint.SetOnDisconnectedHandler(func(err error) {
		msg := "disconnected"
		if err != nil {
			msg = err.Error()
		}
		recordReportError(conn.rule.LoadpointId, msg)
	})
	endpoint.SetOnReconnectedHandler(func() {
		clearReportError(conn.rule.LoadpointId)
		conn.mu.Lock()
		conn.booted = false // re-boot after reconnect
		conn.mu.Unlock()
		// re-drive desired session state onto the new connection - a
		// transaction id from before the drop is deliberately kept (not
		// cleared): OCPP transactions are backend-tracked state, not tied to
		// the WebSocket connection's lifetime, and most Central Systems
		// (incl. SteVe, verified e2e) keep a transaction open across a brief
		// reconnect - starting a second transaction for one continuous
		// physical charge would fragment the billing record the reconnect
		// is trying to preserve. If sessionActive is true but transactionId
		// is nil (the original StartTransaction never got confirmed before
		// the drop), reconcile below issues it fresh.
		conn.enqueue(conn.reconcile)
	})

	go func() {
		for err := range cp.Errors() {
			reportLog.DEBUG.Printf("%s: %v", conn.title, err)
		}
	}()

	conn.mu.Lock()
	conn.restoreLocked()
	conn.mu.Unlock()

	return conn
}

// run dials the connection (with backoff) and drains its job queue. Runs
// until close() is called.
func (conn *reportConnection) run() {
	go conn.dial()

	heartbeat := time.NewTicker(defaultHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case job := <-conn.jobs:
			job()
		case <-heartbeat.C:
			conn.enqueue(conn.sendHeartbeat)
		case <-conn.done:
			return
		}
	}
}

func (conn *reportConnection) sendHeartbeat() {
	if !conn.cp.IsConnected() {
		return
	}
	if _, err := conn.cp.Heartbeat(); err != nil {
		reportLog.DEBUG.Printf("%s: heartbeat: %v", conn.title, err)
	}
}

// statusNow is the connector status implied by the session state
func (conn *reportConnection) statusNow() core.ChargePointStatus {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.sessionActive {
		return core.ChargePointStatusCharging
	}
	return core.ChargePointStatusAvailable
}

// sendStatus reports the connector status. Only after a successful boot, since a
// Central System rejects or ignores status from an unregistered station.
func (conn *reportConnection) sendStatus(status core.ChargePointStatus) {
	conn.mu.Lock()
	booted := conn.booted
	conn.mu.Unlock()
	if !booted || !conn.cp.IsConnected() {
		return
	}
	if _, err := conn.cp.StatusNotification(1, core.NoError, status); err != nil {
		reportLog.DEBUG.Printf("%s: status notification: %v", conn.title, err)
	}
}

// noteIdTagStatus records the idTag status the upstream answered with. It is shown
// in the report status and logged, but never enforced: the charger authorizes the
// idTag itself, and this client has no way to stop it.
func (conn *reportConnection) noteIdTagStatus(status types.AuthorizationStatus) {
	conn.mu.Lock()
	conn.idTagStatus = string(status)
	conn.mu.Unlock()

	if status != types.AuthorizationStatusAccepted {
		reportLog.WARN.Printf("%s: upstream idTag status %s", conn.title, status)
	}
	notifyReportUpdated()
}

func (conn *reportConnection) dial() {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // retry forever
	bo.MaxInterval = 5 * time.Minute

	op := func() error {
		select {
		case <-conn.done:
			return backoff.Permanent(errors.New("closed"))
		default:
		}
		if err := conn.cp.Start(conn.rule.UpstreamURL); err != nil {
			recordReportError(conn.rule.LoadpointId, err.Error())
			return err
		}
		// always notify on a successful connect, even if no prior error was
		// recorded (e.g. the very first attempt succeeds) - the UI still
		// needs to see the connected transition
		reportMu.Lock()
		delete(reportErrors, conn.rule.LoadpointId)
		reportMu.Unlock()
		notifyReportUpdated()
		// flush any session state that accumulated while offline (e.g. a
		// session that started and stopped entirely before the first
		// successful connect - see reconcile())
		conn.enqueue(conn.reconcile)
		return nil
	}

	_ = backoff.Retry(op, bo)
}

func (conn *reportConnection) close() {
	close(conn.done)
	conn.cp.Stop()
}

// enqueue schedules a job on the connection's single worker goroutine, so
// StartTransaction/MeterValues/StopTransaction for one session stay ordered.
// Dropped if the queue is full (backpressure - a report connection that's
// badly behind shouldn't block the loadpoint's own control loop), but this is
// safe for reconcile jobs specifically: they read live desired state off the
// connection rather than closing over a point-in-time value, so a dropped
// reconcile is simply superseded by the next one (triggered by the next
// meter update, session boundary, or reconnect) - nothing is lost, only
// coalesced.
func (conn *reportConnection) enqueue(job func()) {
	select {
	case conn.jobs <- job:
	default:
		reportLog.WARN.Printf("%s: report queue full, dropping message", conn.title)
	}
}

// ensureBoot sends BootNotification once per connection generation. Callers
// must only invoke this while connected (reconcile checks IsConnected first;
// OnTriggerMessage's caller is inherently connected, since the trigger itself
// arrived over the connection) - otherwise the call fails and, since nothing
// retries a bare ensureBoot failure on its own, booted would stay false
// forever. reconcile is what provides the retry, by re-running on every
// (re)connect and dropped-job fallback.
func (conn *reportConnection) ensureBoot() {
	conn.mu.Lock()
	booted := conn.booted
	conn.mu.Unlock()
	if booted {
		return
	}
	if _, err := conn.cp.BootNotification("evcc", "evcc.io"); err != nil {
		reportLog.DEBUG.Printf("%s: boot notification: %v", conn.title, err)
		return
	}
	conn.mu.Lock()
	conn.booted = true
	conn.mu.Unlock()

	conn.sendStatus(conn.statusNow())
}

// maxQueuedEvents bounds the in-memory event queue; meter samples are superseded
// by newer ones, so they are dropped first when the queue is full
const maxQueuedEvents = 1000

type reportEventKind int

const (
	reportEventStart reportEventKind = iota
	reportEventMeter
	reportEventStop
)

// reportEvent is one session boundary or meter sample, stamped with the time it
// happened so that events replayed after an outage keep their original times
type reportEvent struct {
	kind reportEventKind
	at   time.Time
	wh   float64
}

// push appends an event, dropping the oldest meter sample (or the oldest event
// if there is none) when the queue is full. Caller holds conn.mu.
func (conn *reportConnection) push(ev reportEvent) {
	if len(conn.events) >= maxQueuedEvents {
		idx := 0
		for i, e := range conn.events {
			if e.kind == reportEventMeter {
				idx = i
				break
			}
		}
		if conn.events[idx].kind != reportEventMeter {
			reportLog.WARN.Printf("%s: report queue full, dropping oldest event", conn.title)
		}
		conn.events = append(conn.events[:idx], conn.events[idx+1:]...)
	}
	if ev.kind != reportEventStop {
		conn.lastWh = ev.wh
		conn.lastAt = ev.at
	}
	conn.events = append(conn.events, ev)

	if ev.kind != reportEventMeter {
		conn.persistLocked()
	}
}

// reconcile drains the event queue in order: Authorize and StartTransaction for a
// start, MeterValues for samples, StopTransaction for a stop. It stops at the first
// failure and keeps the remaining events for the next call, and it never attempts
// anything while disconnected, so a call that's certain to fail doesn't waste the
// one attempt an unconditional fire-and-forget job gets.
func (conn *reportConnection) reconcile() {
	if !conn.cp.IsConnected() {
		return
	}

	conn.ensureBoot()

	for {
		conn.mu.Lock()
		if len(conn.events) == 0 {
			conn.mu.Unlock()
			return
		}
		ev := conn.events[0]
		var next *reportEvent
		if len(conn.events) > 1 {
			n := conn.events[1]
			next = &n
		}
		txID := conn.transactionId
		conn.mu.Unlock()

		var err error
		switch ev.kind {
		case reportEventStart:
			err = conn.sendStart(ev)
		case reportEventMeter:
			err = conn.sendMeter(ev, next, txID)
		case reportEventStop:
			err = conn.sendStop(ev, txID)
		}

		if err != nil {
			return
		}

		conn.mu.Lock()
		conn.events = conn.events[1:]
		if ev.kind != reportEventMeter {
			conn.persistLocked()
		}
		conn.mu.Unlock()
	}
}

func (conn *reportConnection) sendStart(ev reportEvent) error {
	idTag := conn.rule.IdTag

	auth, err := conn.cp.Authorize(idTag)
	if err != nil {
		reportLog.DEBUG.Printf("%s: authorize: %v", conn.title, err)
		return err
	}
	if auth.IdTagInfo != nil {
		conn.noteIdTagStatus(auth.IdTagInfo.Status)
	}

	res, err := conn.cp.StartTransaction(1, idTag, int(ev.wh), types.NewDateTime(ev.at))
	if err != nil {
		reportLog.DEBUG.Printf("%s: start transaction: %v", conn.title, err)
		return err
	}
	if res.IdTagInfo != nil {
		conn.noteIdTagStatus(res.IdTagInfo.Status)
	}

	conn.mu.Lock()
	conn.transactionId = &res.TransactionId
	conn.persistLocked()
	conn.mu.Unlock()

	// don't let a previous session's send time throttle this new session's first sample
	conn.lastMeterSent = time.Time{}
	return nil
}

// sendMeter sends a sample unless a newer one follows directly, or it falls inside
// the throttle interval and nothing else is queued behind it. The sample waiting
// for the interval is kept at the head of the queue and sent by a later call.
func (conn *reportConnection) sendMeter(ev reportEvent, next *reportEvent, txID *int) error {
	if txID == nil {
		return nil
	}

	if next != nil && next.kind == reportEventMeter {
		return nil
	}

	now := time.Now()
	if next == nil && now.Sub(conn.lastMeterSent) < conn.meterInterval {
		return errReportThrottled
	}

	mv := types.MeterValue{
		Timestamp: types.NewDateTime(ev.at),
		SampledValue: []types.SampledValue{{
			Value:     fmt.Sprintf("%.0f", ev.wh),
			Measurand: types.MeasurandEnergyActiveImportRegister,
			Unit:      types.UnitOfMeasureWh,
		}},
	}
	if _, err := conn.cp.MeterValues(1, []types.MeterValue{mv}, func(r *core.MeterValuesRequest) {
		r.TransactionId = txID
	}); err != nil {
		reportLog.DEBUG.Printf("%s: meter values: %v", conn.title, err)
		return err
	}

	conn.lastMeterSent = now

	conn.mu.Lock()
	conn.persistLocked()
	conn.mu.Unlock()
	return nil
}

func (conn *reportConnection) sendStop(ev reportEvent, txID *int) error {
	if txID == nil {
		return nil
	}

	if _, err := conn.cp.StopTransaction(int(ev.wh), types.NewDateTime(ev.at), *txID); err != nil {
		reportLog.DEBUG.Printf("%s: stop transaction: %v", conn.title, err)
		return err
	}

	conn.mu.Lock()
	conn.sessionActive = false
	conn.transactionId = nil
	conn.mu.Unlock()
	return nil
}

// errReportThrottled keeps a throttled sample at the head of the queue
var errReportThrottled = errors.New("throttled")

// SessionReporter forwards loadpoint sessions to the report connections. It is wired
// into core at boot, so core only depends on the interface it declares.
type SessionReporter struct{}

func (SessionReporter) SessionStart(loadpointId int, meterStartWh float64) {
	ReportSessionStart(loadpointId, meterStartWh)
}

func (SessionReporter) SessionMeter(loadpointId int, registerWh float64) {
	ReportMeterValue(loadpointId, registerWh)
}

func (SessionReporter) SessionStop(loadpointId int, meterStopWh float64) {
	ReportSessionStop(loadpointId, meterStopWh)
}

// ReportSessionStart notifies the loadpoint's report connection (if any) that
// a charging session started. No-op if the loadpoint has no rule configured.
func ReportSessionStart(loadpointId int, meterStartWh float64) {
	reportMu.RLock()
	conn := connections[loadpointId]
	reportMu.RUnlock()
	if conn == nil {
		return
	}

	conn.mu.Lock()
	conn.sessionActive = true
	conn.push(reportEvent{kind: reportEventStart, at: time.Now(), wh: meterStartWh})
	conn.mu.Unlock()

	conn.enqueue(conn.reconcile)
	conn.enqueue(func() { conn.sendStatus(core.ChargePointStatusCharging) })
}

// ReportMeterValue notifies the loadpoint's report connection (if any) of the
// current cumulative session energy. No-op without an active reported session.
func ReportMeterValue(loadpointId int, energyWh float64) {
	reportMu.RLock()
	conn := connections[loadpointId]
	reportMu.RUnlock()
	if conn == nil {
		return
	}

	conn.mu.Lock()
	if !conn.sessionActive {
		conn.mu.Unlock()
		return
	}
	conn.push(reportEvent{kind: reportEventMeter, at: time.Now(), wh: energyWh})
	conn.mu.Unlock()

	conn.enqueue(conn.reconcile)
}

// ReportSessionStop notifies the loadpoint's report connection (if any) that
// the charging session ended. Queued behind any pending samples, so the stop
// carries the last value reported before it.
func ReportSessionStop(loadpointId int, meterStopWh float64) {
	reportMu.RLock()
	conn := connections[loadpointId]
	reportMu.RUnlock()
	if conn == nil {
		return
	}

	conn.mu.Lock()
	if !conn.sessionActive {
		conn.mu.Unlock()
		return
	}
	conn.sessionActive = false
	conn.push(reportEvent{kind: reportEventStop, at: time.Now(), wh: meterStopWh})
	conn.mu.Unlock()

	conn.enqueue(conn.reconcile)
	conn.enqueue(func() { conn.sendStatus(core.ChargePointStatusAvailable) })
}

// reportHandler implements the inbound OCPP call handlers (remote control)
// for one report connection.
type reportHandler struct {
	conn *reportConnection
}

var _ core.ChargePointHandler = (*reportHandler)(nil)
var _ remotetrigger.ChargePointHandler = (*reportHandler)(nil)

// OnRemoteStartTransaction / OnRemoteStopTransaction: the report client is a
// one-way reporter (evcc-io/evcc#32989 - "no need to support locking/unlocking
// the charger, just logging the charge sessions would be sufficient"). Its
// upstream is a billing/certification backend for a charger evcc already
// fully controls itself, not an access-control authority for it - unlike the
// forwarder's upstream, which can legitimately be one. So these are honestly
// rejected rather than translated into loadpoint control.
func (h *reportHandler) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}

func (h *reportHandler) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}

// OnUnlockConnector: evcc has no generic cross-charger unlock capability, so
// this is an honest rejection rather than a fake success.
func (h *reportHandler) OnUnlockConnector(request *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusNotSupported), nil
}

// OnReset: no generic per-charger reset exists, and resetting evcc itself
// would be wrong. Benign no-op accept so upstream doesn't flag the station
// as faulty.
func (h *reportHandler) OnReset(request *core.ResetRequest) (*core.ResetConfirmation, error) {
	reportLog.DEBUG.Printf("%s: reset requested (no-op)", h.conn.title)
	return core.NewResetConfirmation(core.ResetStatusAccepted), nil
}

func (h *reportHandler) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusAccepted), nil
}

func (h *reportHandler) OnChangeConfiguration(request *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusNotSupported), nil
}

func (h *reportHandler) OnClearCache(request *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	return core.NewClearCacheConfirmation(core.ClearCacheStatusAccepted), nil
}

func (h *reportHandler) OnDataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	return core.NewDataTransferConfirmation(core.DataTransferStatusUnknownVendorId), nil
}

func (h *reportHandler) OnGetConfiguration(request *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	return core.NewGetConfigurationConfirmation(nil), nil
}

// OnTriggerMessage re-emits the requested message from currently cached
// state, best-effort.
func (h *reportHandler) OnTriggerMessage(request *remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error) {
	switch request.RequestedMessage {
	case core.BootNotificationFeatureName:
		h.conn.mu.Lock()
		h.conn.booted = false
		h.conn.mu.Unlock()
		h.conn.enqueue(h.conn.ensureBoot)
	case core.MeterValuesFeatureName:
		h.conn.mu.Lock()
		if h.conn.sessionActive {
			h.conn.push(reportEvent{kind: reportEventMeter, at: time.Now(), wh: h.conn.lastWh})
		}
		h.conn.mu.Unlock()
		h.conn.enqueue(h.conn.reconcile)
	case core.StatusNotificationFeatureName:
		h.conn.enqueue(func() { h.conn.sendStatus(h.conn.statusNow()) })
	case core.HeartbeatFeatureName:
		h.conn.enqueue(h.conn.sendHeartbeat)
	default:
		return remotetrigger.NewTriggerMessageConfirmation(remotetrigger.TriggerMessageStatusNotImplemented), nil
	}

	return remotetrigger.NewTriggerMessageConfirmation(remotetrigger.TriggerMessageStatusAccepted), nil
}

// ReportEnabledSetting returns the saved master switch, defaulting to true when it
// was never set
func ReportEnabledSetting() bool {
	b, err := settings.Bool(keys.OcppReportEnabled)
	if err != nil {
		return true
	}
	return b
}
