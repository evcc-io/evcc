package ocpp

// Hybrid OCPP proxy. Chargers connect to evcc's central system; for each charger
// with a matching rule a "sidecar" WebSocket to the upstream OCPP server runs in
// parallel. Billing-critical Calls (actionsRelayedToUpstream) are relayed to
// upstream as the authoritative responder and evcc's handler is bypassed; all
// other messages are processed by evcc and mirrored to upstream for observation.
// Upstream Calls are injected into the charger unless the rule is read-only.
//
// Design: docs/agents/ocpp-forwarder.md

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

// forwarder rules, guarded by forwarderMu
var (
	forwarderMu    sync.RWMutex
	forwarderRules []ForwarderRule
)

// ForwarderEnabled returns true when at least one rule is configured.
func ForwarderEnabled() bool {
	forwarderMu.RLock()
	defer forwarderMu.RUnlock()
	return len(forwarderRules) > 0
}

// ApplyForwarderRules replaces the forwarding rules at runtime and republishes status.
func ApplyForwarderRules(rules []ForwarderRule) {
	forwarderMu.Lock()
	forwarderRules = rules
	forwarderMu.Unlock()

	valid := make(map[string]bool, len(rules))
	hasWildcard := false
	for _, r := range rules {
		valid[r.StationID] = true
		if r.StationID == "*" {
			hasWildcard = true
		}
	}

	// drop sidecars/errors for removed rules
	var stale []*sidecar
	sidecarsMu.Lock()
	for id, sc := range sidecars {
		if !valid[id] && !hasWildcard {
			stale = append(stale, sc)
			delete(sidecars, id)
		}
	}
	for id := range forwarderErrors {
		if !valid[id] && !hasWildcard {
			delete(forwarderErrors, id)
		}
	}
	sidecarsMu.Unlock()
	for _, sc := range stale {
		sc.conn.CloseNow()
	}

	// connected charger: (re)dial sidecar; otherwise test-dial to surface config errors
	for _, r := range rules {
		if r.StationID == "*" || r.UpstreamURL == "" {
			continue
		}
		sidecarsMu.Lock()
		sc, active := sidecars[r.StationID]
		connected := connectedChargers[r.StationID]
		var changed *sidecar
		if active && !sc.rule.sameConnection(r) {
			changed = sc
			delete(sidecars, r.StationID)
			active = false
		}
		sidecarsMu.Unlock()
		if changed != nil {
			changed.conn.CloseNow()
		}
		if active {
			continue
		}
		if connected {
			pendingMu.Lock()
			pendingMsgs[r.StationID] = nil
			pendingMu.Unlock()
			go runUpstreamSidecar(r.StationID, r)
		} else {
			go validateUpstream(r.StationID, r)
		}
	}

	notifyUpdated()
}

// dialUpstream opens a websocket connection to the rule's upstream server for the given charger.
func dialUpstream(id string, rule ForwarderRule) (*websocket.Conn, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: rule.Insecure}
	if rule.CaCert != "" {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(rule.CaCert)); !ok {
			return nil, errors.New("invalid CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	var header http.Header
	if rule.Username != "" || rule.Password != "" {
		header = authHeader(rule.Username, rule.Password)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, strings.TrimRight(rule.UpstreamURL, "/")+rule.upstreamPath(id), &websocket.DialOptions{
		Subprotocols: []string{"ocpp1.6"},
		HTTPHeader:   header,
		HTTPClient:   &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}},
	})
	return conn, err
}

// validateUpstream test-dials a rule's upstream and records/clears the error so
// the UI reflects unreachable hosts.
func validateUpstream(id string, rule ForwarderRule) {
	conn, err := dialUpstream(id, rule)

	// a charger may have connected meanwhile; its sidecar is authoritative
	sidecarsMu.Lock()
	_, active := sidecars[id]
	sidecarsMu.Unlock()
	if active {
		if conn != nil {
			conn.CloseNow()
		}
		return
	}

	if err != nil {
		recordForwarderError(id, err.Error())
		notifyUpdated()
		return
	}
	conn.Close(websocket.StatusNormalClosure, "")
	if clearForwarderError(id) {
		notifyUpdated()
	}
}

// ForwarderRules returns the current forwarding rules.
func ForwarderRules() []ForwarderRule {
	forwarderMu.RLock()
	defer forwarderMu.RUnlock()
	return forwarderRules
}

// actionsRelayedToUpstream lists actions for which upstream is the authoritative
// Central System: evcc's handler is bypassed and upstream's reply relayed back.
var actionsRelayedToUpstream = map[string]bool{
	"Authorize":        true,
	"StartTransaction": true,
	"StopTransaction":  true,
	"DataTransfer":     true,
}

var forwarderLog = util.NewLogger("ocpp-forwarder")

// init wires the forwarder hooks declared in instance.go.
func init() {
	chargerConnectHook = onChargerConnect
	chargerDisconnectHook = onChargerDisconnect
	chargerMessageHook = onChargerMessage
}

// sidecar holds an upstream connection for a single charger.
type sidecar struct {
	chargerID string
	rule      ForwarderRule // rule used to dial; detects param changes
	conn      *websocket.Conn

	// message IDs of upstream-initiated Calls; the charger's reply is routed back to upstream
	pendingUpstreamCallsMu sync.Mutex
	pendingUpstreamCalls   map[string]struct{}

	// message IDs of charger Calls whose evcc handler was bypassed; upstream's reply is relayed to the charger
	pendingChargerCallsMu sync.Mutex
	pendingChargerCalls   map[string]struct{}

	// when > 0, forward at most one MeterValues per interval to upstream; evcc still sees every frame
	meterInterval  time.Duration
	lastMeterFwdMu sync.Mutex
	lastMeterFwd   time.Time
}

var (
	sidecarsMu sync.Mutex
	sidecars   = make(map[string]*sidecar)

	// connected charger ids so a runtime-added rule can start a sidecar; guarded by sidecarsMu
	connectedChargers = make(map[string]bool)

	// last upstream failure per charger, surfaced to the UI; guarded by sidecarsMu
	forwarderErrors = make(map[string]string)

	// raw frames buffered per charger while the sidecar dials; flushed in order
	// on connect so BootNotification reaches upstream
	pendingMu   sync.Mutex
	pendingMsgs = make(map[string][][]byte)

	// last BootNotification per connected charger, replayed to upstream when a
	// sidecar (re)connects mid-session so upstream sees a boot before transactions
	bootMu   sync.Mutex
	lastBoot = make(map[string][]byte)
)

// reconnect backoff bounds; variables for testing
var (
	reconnectInitialDelay = 5 * time.Second
	reconnectMaxDelay     = 5 * time.Minute
)

// runUpstreamSidecar dials a charger's upstream sidecar and re-dials with
// exponential backoff for as long as the charger stays connected and the rule
// remains in effect.
func runUpstreamSidecar(id string, rule ForwarderRule) {
	bo := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(reconnectInitialDelay),
		backoff.WithMaxInterval(reconnectMaxDelay),
		backoff.WithMaxElapsedTime(0),
	)

	for {
		if !dialUpstreamSidecar(id, rule) {
			delay := bo.NextBackOff()
			forwarderLog.DEBUG.Printf("forwarder: reconnecting upstream for %s in %v", id, delay)
			time.Sleep(delay)
			continue
		}

		bo.Reset()

		if current, ok := resolveRule(id); !ok || current.UpstreamURL == "" || !current.sameConnection(rule) {
			return
		}

		sidecarsMu.Lock()
		connected := connectedChargers[id]
		_, active := sidecars[id]
		sidecarsMu.Unlock()

		if !connected || active {
			return
		}

		pendingMu.Lock()
		pendingMsgs[id] = nil
		pendingMu.Unlock()
	}
}

// resolveRule returns the forwarding rule for chargerID, or the "*" fallback.
func resolveRule(chargerID string) (ForwarderRule, bool) {
	forwarderMu.RLock()
	rules := forwarderRules
	forwarderMu.RUnlock()
	var fallback ForwarderRule
	var hasFallback bool
	for _, r := range rules {
		if r.StationID == chargerID {
			return r, true
		}
		if r.StationID == "*" {
			fallback = r
			hasFallback = true
		}
	}
	return fallback, hasFallback
}

// onChargerConnect dials a sidecar for the connecting charger when a rule matches.
func onChargerConnect(ch ws.Channel) {
	id := ch.ID()

	sidecarsMu.Lock()
	connectedChargers[id] = true
	sidecarsMu.Unlock()

	rule, ok := resolveRule(id)
	if !ok {
		return
	}
	// open the buffer before dialling so early messages are captured, not dropped
	pendingMu.Lock()
	pendingMsgs[id] = nil
	pendingMu.Unlock()

	go runUpstreamSidecar(id, rule)
}

func dialUpstreamSidecar(id string, rule ForwarderRule) bool {
	conn, err := dialUpstream(id, rule)
	if err != nil {
		forwarderLog.WARN.Printf("forwarder: dial upstream for %s: %v", id, err)
		recordForwarderError(id, err.Error())
		notifyUpdated()
		drainPendingWithErrors(id, nil)
		return false
	}
	conn.SetReadLimit(-1) // no limit; OCPP frames can be large

	sc := &sidecar{
		chargerID:            id,
		rule:                 rule,
		conn:                 conn,
		pendingUpstreamCalls: make(map[string]struct{}),
		pendingChargerCalls:  make(map[string]struct{}),
	}

	// install sidecar and drain the pending buffer
	sidecarsMu.Lock()
	pendingMu.Lock()
	old := sidecars[id]
	sidecars[id] = sc
	buffered := pendingMsgs[id]
	delete(pendingMsgs, id)
	pendingMu.Unlock()
	sidecarsMu.Unlock()

	// a concurrent dial may have installed a sidecar meanwhile; this one wins
	if old != nil {
		old.conn.CloseNow()
	}

	// replay the charger's BootNotification (with a fresh message id) when the
	// buffer doesn't already carry one, i.e. the sidecar (re)connects mid-session;
	// upstream's reply is discarded as evcc has long answered the charger
	if !slices.ContainsFunc(buffered, func(frame []byte) bool {
		msgType, _, action, err := parseOCPPFrame(frame)
		return err == nil && msgType == ocppj.CALL && action == "BootNotification"
	}) {
		bootMu.Lock()
		boot := lastBoot[id]
		bootMu.Unlock()
		if boot != nil {
			if frame, err := withMessageID(boot, fmt.Sprintf("evcc-boot-%d", time.Now().UnixNano())); err == nil {
				buffered = append([][]byte{frame}, buffered...)
			}
		}
	}

	// flush buffered Calls; register relay actions so upstream's reply routes back.
	// CallResults/Errors are skipped: evcc already answered them.
	flushed := 0
	for _, frame := range buffered {
		msgType, msgID, action, err := parseOCPPFrame(frame)
		if err != nil || msgType != ocppj.CALL {
			continue
		}
		if actionsRelayedToUpstream[action] {
			sc.pendingChargerCallsMu.Lock()
			sc.pendingChargerCalls[msgID] = struct{}{}
			sc.pendingChargerCallsMu.Unlock()
		}
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		writeErr := conn.Write(writeCtx, websocket.MessageText, frame)
		writeCancel()
		if writeErr != nil {
			forwarderLog.ERROR.Printf("forwarder: write buffered frame to upstream for %s: %v", id, writeErr)
			conn.CloseNow()
			sidecarsMu.Lock()
			if sidecars[id] == sc {
				delete(sidecars, id)
			}
			sidecarsMu.Unlock()
			recordForwarderError(id, writeErr.Error())
			notifyUpdated()
			return false
		}
		flushed++
	}
	if flushed > 0 {
		forwarderLog.DEBUG.Printf("forwarder: flushed %d call(s) to upstream for %s (skipped %d non-call frame(s))",
			flushed, id, len(buffered)-flushed)
	}

	clearForwarderError(id)
	notifyUpdated()
	forwarderLog.INFO.Printf("forwarder: %s → %s", id, strings.TrimRight(rule.UpstreamURL, "/")+rule.upstreamPath(id))

	sc.readFromUpstream()

	return true
}

// drainPendingWithErrors discards charger id's pending buffer, sending a CallError
// to the charger for each buffered relay Call so it is not left hanging. sc may be nil.
func drainPendingWithErrors(id string, sc *sidecar) {
	pendingMu.Lock()
	buffered := pendingMsgs[id]
	delete(pendingMsgs, id)
	pendingMu.Unlock()

	cs, err := Instance()
	if err != nil {
		return
	}

	for _, frame := range buffered {
		msgType, msgID, action, err := parseOCPPFrame(frame)
		if err != nil || msgType != ocppj.CALL || !actionsRelayedToUpstream[action] {
			continue
		}
		errFrame, _ := (&ocppj.CallError{
			MessageTypeId:    ocppj.CALL_ERROR,
			UniqueId:         msgID,
			ErrorCode:        ocppj.GenericError,
			ErrorDescription: "Upstream OCPP server unavailable",
		}).MarshalJSON()
		if writeErr := cs.Write(id, errFrame); writeErr != nil {
			forwarderLog.WARN.Printf("forwarder: send error for pending call %s to %s: %v", msgID, id, writeErr)
		}
	}

	// error out tracked pendingChargerCalls (sidecar dropped mid-session)
	if sc != nil {
		sc.pendingChargerCallsMu.Lock()
		for msgID := range sc.pendingChargerCalls {
			errFrame, _ := (&ocppj.CallError{
				MessageTypeId:    ocppj.CALL_ERROR,
				UniqueId:         msgID,
				ErrorCode:        ocppj.GenericError,
				ErrorDescription: "Upstream OCPP server disconnected",
			}).MarshalJSON()
			if writeErr := cs.Write(sc.chargerID, errFrame); writeErr != nil {
				forwarderLog.WARN.Printf("forwarder: send disconnect error for %s to %s: %v", msgID, sc.chargerID, writeErr)
			}
		}
		sc.pendingChargerCalls = make(map[string]struct{})
		sc.pendingChargerCallsMu.Unlock()
	}
}

// onChargerDisconnect closes the charger's sidecar connection.
func onChargerDisconnect(ch ws.Channel) {
	id := ch.ID()

	// discard pending buffer and cached boot frame
	pendingMu.Lock()
	delete(pendingMsgs, id)
	pendingMu.Unlock()
	bootMu.Lock()
	delete(lastBoot, id)
	bootMu.Unlock()

	sidecarsMu.Lock()
	delete(connectedChargers, id)
	sc, ok := sidecars[id]
	if ok {
		delete(sidecars, id)
	}
	_, hadErr := forwarderErrors[id]
	delete(forwarderErrors, id)
	sidecarsMu.Unlock()
	if ok {
		sc.conn.CloseNow()
		forwarderLog.DEBUG.Printf("forwarder: %s upstream connection closed", id)
	}
	if ok || hadErr {
		notifyUpdated()
	}
}

// onChargerMessage handles a raw OCPP frame from a charger and reports whether
// evcc's handler should be bypassed (upstream is the authoritative responder).
// Relay-action Calls are forwarded and bypassed; other Calls are forwarded and
// also handled by evcc; CallResults/Errors are forwarded only when they answer
// an upstream-initiated Call.
func onChargerMessage(ch ws.Channel, data []byte) bool {
	id := ch.ID()

	msgType, msgID, action, err := parseOCPPFrame(data)
	if err != nil {
		return false
	}

	// remember the charger's boot frame for replay on sidecar (re)connect
	if msgType == ocppj.CALL && action == "BootNotification" {
		bootMu.Lock()
		lastBoot[id] = slices.Clone(data)
		bootMu.Unlock()
	}

	sidecarsMu.Lock()
	sc := sidecars[id]
	sidecarsMu.Unlock()

	switch msgType {
	case ocppj.CALL:
		relay := actionsRelayedToUpstream[action]

		if sc != nil {
			// throttle MeterValues to upstream; evcc still processes every frame
			if action == "MeterValues" && sc.meterInterval > 0 {
				sc.lastMeterFwdMu.Lock()
				elapsed := time.Since(sc.lastMeterFwd)
				if elapsed < sc.meterInterval {
					sc.lastMeterFwdMu.Unlock()
					return false // evcc handles normally; skip upstream
				}
				sc.lastMeterFwd = time.Now()
				sc.lastMeterFwdMu.Unlock()
			}
			if relay {
				sc.pendingChargerCallsMu.Lock()
				sc.pendingChargerCalls[msgID] = struct{}{}
				sc.pendingChargerCallsMu.Unlock()
			}
			if err := sc.conn.Write(context.Background(), websocket.MessageText, data); err != nil {
				forwarderLog.ERROR.Printf("forwarder: write to upstream for %s: %v", id, err)
			}
			return relay
		}

		// sidecar not ready: buffer if a pending slot exists
		pendingMu.Lock()
		_, hasPending := pendingMsgs[id]
		if hasPending {
			pendingMsgs[id] = append(pendingMsgs[id], slices.Clone(data))
		}
		pendingMu.Unlock()

		// bypass evcc for relay actions while buffering; upstream answers after flush
		return relay && hasPending

	case ocppj.CALL_RESULT, ocppj.CALL_ERROR:
		// forward only if this answers an upstream-initiated Call
		if sc == nil {
			return false
		}
		sc.pendingUpstreamCallsMu.Lock()
		_, isUpstream := sc.pendingUpstreamCalls[msgID]
		if isUpstream {
			delete(sc.pendingUpstreamCalls, msgID)
		}
		sc.pendingUpstreamCallsMu.Unlock()
		if !isUpstream {
			return false
		}
		if err := sc.conn.Write(context.Background(), websocket.MessageText, data); err != nil {
			forwarderLog.ERROR.Printf("forwarder: write upstream reply for %s: %v", id, err)
		}
		return false // evcc may also see it; harmless for unknown IDs
	}
	return false
}

// readFromUpstream relays frames from upstream: Calls are injected into the
// charger (reply routed back), responses to bypassed charger Calls are relayed
// to the charger, others discarded. Calls are rejected in read-only mode.
func (sc *sidecar) readFromUpstream() {
	defer func() {
		sidecarsMu.Lock()
		if sidecars[sc.chargerID] == sc {
			delete(sidecars, sc.chargerID)
		}
		sidecarsMu.Unlock()
		// error out pending relayed calls; upstream is gone
		drainPendingWithErrors(sc.chargerID, sc)
		sc.conn.CloseNow()
		notifyUpdated()
	}()

	cs, err := Instance()
	if err != nil {
		forwarderLog.ERROR.Printf("forwarder: central system unavailable for %s: %v", sc.chargerID, err)
		return
	}

	for {
		_, msg, err := sc.conn.Read(context.Background())
		if err != nil {
			forwarderLog.DEBUG.Printf("forwarder: upstream disconnected for %s: %v", sc.chargerID, err)
			// charger disconnect deletes the sidecar first, so a still-current sidecar means upstream dropped
			sidecarsMu.Lock()
			upstreamDrop := sidecars[sc.chargerID] == sc
			sidecarsMu.Unlock()
			if upstreamDrop {
				recordForwarderError(sc.chargerID, err.Error())
			}
			return
		}

		msgType, msgID, action, err := parseOCPPFrame(msg)
		if err != nil {
			forwarderLog.WARN.Printf("forwarder: upstream parse error for %s: %v", sc.chargerID, err)
			continue
		}

		switch msgType {
		case ocppj.CALL:
			// resolve the current rule so a runtime ReadOnly toggle applies live
			if rule, ok := resolveRule(sc.chargerID); ok && rule.ReadOnly {
				forwarderLog.DEBUG.Printf("forwarder: blocking upstream call %s in read-only session %s", msgID, sc.chargerID)
				errFrame, _ := (&ocppj.CallError{
					MessageTypeId:    ocppj.CALL_ERROR,
					UniqueId:         msgID,
					ErrorCode:        ocppj.SecurityError,
					ErrorDescription: "Charger control not allowed: forwarder is in read-only mode",
				}).MarshalJSON()
				_ = sc.conn.Write(context.Background(), websocket.MessageText, errFrame)
				continue
			}

			// absorb upstream's MeterValueSampleInterval as our throttle; reply
			// Accepted without touching the charger's config
			if action == "ChangeConfiguration" {
				if interval, ok := extractMeterValueSampleInterval(msg); ok {
					sc.lastMeterFwdMu.Lock()
					sc.meterInterval = interval
					sc.lastMeterFwdMu.Unlock()
					forwarderLog.DEBUG.Printf("forwarder: upstream set MeterValueSampleInterval=%v for %s", interval, sc.chargerID)
					accepted, _ := (&ocppj.CallResult{
						MessageTypeId: ocppj.CALL_RESULT,
						UniqueId:      msgID,
						Payload:       core.NewChangeConfigurationConfirmation(core.ConfigurationStatusAccepted),
					}).MarshalJSON()
					_ = sc.conn.Write(context.Background(), websocket.MessageText, accepted)
					continue
				}
			}

			// track id so the charger's reply is forwarded back to upstream
			sc.pendingUpstreamCallsMu.Lock()
			sc.pendingUpstreamCalls[msgID] = struct{}{}
			sc.pendingUpstreamCallsMu.Unlock()

			if err := cs.Write(sc.chargerID, msg); err != nil {
				forwarderLog.ERROR.Printf("forwarder: inject upstream call into charger %s: %v", sc.chargerID, err)
			}

		case ocppj.CALL_RESULT, ocppj.CALL_ERROR:
			// upstream's authoritative response to a bypassed charger Call?
			sc.pendingChargerCallsMu.Lock()
			_, isChargerCall := sc.pendingChargerCalls[msgID]
			if isChargerCall {
				delete(sc.pendingChargerCalls, msgID)
			}
			sc.pendingChargerCallsMu.Unlock()

			if isChargerCall {
				// relay to charger; its handler was bypassed and it awaits this reply
				if err := cs.Write(sc.chargerID, msg); err != nil {
					forwarderLog.ERROR.Printf("forwarder: relay upstream response to charger %s: %v", sc.chargerID, err)
				}
				continue
			}
			// discard: evcc already replied for non-relay Calls
		}
	}
}

// ForwarderSessionStatus is the observable state of one forwarder session.
type ForwarderSessionStatus struct {
	ChargerID         string `json:"chargerId"`
	UpstreamURL       string `json:"upstreamUrl"`
	UpstreamConnected bool   `json:"upstreamConnected"`
	Error             string `json:"error,omitempty"`
}

// recordForwarderError stores a charger's last upstream failure; caller must notifyUpdated.
func recordForwarderError(id, msg string) {
	sidecarsMu.Lock()
	forwarderErrors[id] = msg
	sidecarsMu.Unlock()
}

// clearForwarderError drops a charger's stored error, reporting whether one existed; caller must notifyUpdated.
func clearForwarderError(id string) bool {
	sidecarsMu.Lock()
	_, ok := forwarderErrors[id]
	delete(forwarderErrors, id)
	sidecarsMu.Unlock()
	return ok
}

var (
	forwarderCbMu      sync.Mutex
	forwarderUpdatedCb func()
)

func notifyUpdated() {
	status := GetForwarderStatus()
	forwarderLog.DEBUG.Printf("forwarder: notifyUpdated sessions=%d", len(status))
	forwarderCbMu.Lock()
	cb := forwarderUpdatedCb
	forwarderCbMu.Unlock()
	if cb != nil {
		cb()
	}
}

// SetForwarderUpdated registers a callback fired when a session connects or disconnects.
func SetForwarderUpdated(cb func()) {
	forwarderCbMu.Lock()
	forwarderUpdatedCb = cb
	forwarderCbMu.Unlock()
}

// GetForwarderStatus returns a snapshot of all active forwarder sessions.
func GetForwarderStatus() []ForwarderSessionStatus {
	forwarderMu.RLock()
	rules := append([]ForwarderRule(nil), forwarderRules...)
	forwarderMu.RUnlock()

	sidecarsMu.Lock()
	defer sidecarsMu.Unlock()
	out := make([]ForwarderSessionStatus, 0, len(rules))
	for _, r := range rules {
		if r.StationID == "*" {
			continue
		}
		st := ForwarderSessionStatus{
			ChargerID:   r.StationID,
			UpstreamURL: strings.TrimRight(r.UpstreamURL, "/"),
		}
		if _, ok := sidecars[r.StationID]; ok {
			st.UpstreamConnected = true
		} else if msg, ok := forwarderErrors[r.StationID]; ok {
			st.Error = msg
		}
		out = append(out, st)
	}
	return out
}

// sameConnection reports whether two rules dial upstream identically (no reconnect
// needed). ReadOnly is excluded; it applies live per message.
func (r ForwarderRule) sameConnection(o ForwarderRule) bool {
	return r.UpstreamURL == o.UpstreamURL &&
		r.UpstreamStationID == o.UpstreamStationID &&
		r.Username == o.Username &&
		r.Password == o.Password &&
		r.Insecure == o.Insecure &&
		r.CaCert == o.CaCert
}

// upstreamPath returns the upstream WebSocket path, defaulting to the charger's own ID.
func (r ForwarderRule) upstreamPath(chargerID string) string {
	sid := r.UpstreamStationID
	if sid == "" {
		sid = chargerID
	}
	return "/" + strings.TrimLeft(sid, "/")
}

// authHeader returns a Basic Auth header for the given credentials.
func authHeader(username, password string) http.Header {
	creds := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	h := make(http.Header)
	h.Set("Authorization", "Basic "+creds)
	return h
}

// extractMeterValueSampleInterval returns the interval from a ChangeConfiguration
// Call for key MeterValueSampleInterval.
func extractMeterValueSampleInterval(msg []byte) (time.Duration, bool) {
	var frame []json.RawMessage
	if err := json.Unmarshal(msg, &frame); err != nil || len(frame) < 4 {
		return 0, false
	}
	var req core.ChangeConfigurationRequest
	if err := json.Unmarshal(frame[3], &req); err != nil {
		return 0, false
	}
	if req.Key != "MeterValueSampleInterval" {
		return 0, false
	}
	secs, err := strconv.Atoi(req.Value)
	if err != nil || secs <= 0 {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

// withMessageID returns a copy of a raw OCPP frame with its message id replaced.
func withMessageID(frame []byte, msgID string) ([]byte, error) {
	var parts []json.RawMessage
	if err := json.Unmarshal(frame, &parts); err != nil {
		return nil, err
	}
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid OCPP frame")
	}
	id, err := json.Marshal(msgID)
	if err != nil {
		return nil, err
	}
	parts[1] = id
	return json.Marshal(parts)
}

// parseOCPPFrame extracts the message type, id and (for Calls) action from a raw frame.
func parseOCPPFrame(msg []byte) (msgType ocppj.MessageType, msgID string, action string, err error) {
	var frame []json.RawMessage
	if err = json.Unmarshal(msg, &frame); err != nil || len(frame) < 2 {
		return 0, "", "", fmt.Errorf("invalid OCPP frame")
	}
	if err = json.Unmarshal(frame[0], &msgType); err != nil {
		return 0, "", "", fmt.Errorf("invalid message type: %w", err)
	}
	if err = json.Unmarshal(frame[1], &msgID); err != nil {
		return 0, "", "", fmt.Errorf("invalid message id: %w", err)
	}
	if msgType == ocppj.CALL && len(frame) >= 3 {
		_ = json.Unmarshal(frame[2], &action) // best-effort; empty string if missing
	}
	return msgType, msgID, action, nil
}
