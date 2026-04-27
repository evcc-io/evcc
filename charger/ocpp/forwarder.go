package ocpp

// forwarder.go — OCPP message forwarder/proxy (hybrid relay model).
//
// Chargers connect directly to evcc's OCPP central system on the normal port.
// For each charger with a matching rule a "sidecar" WebSocket connection is
// established to the upstream OCPP server.
//
// Two forwarding modes apply simultaneously:
//
//  1. Transparent relay for billing-critical messages (Authorize, StartTransaction,
//     StopTransaction, DataTransfer):
//       charger → evcc (hook captures, evcc handler bypassed)
//         → forwarded to upstream sidecar
//         ← upstream CallResult/CallError relayed back to charger
//     evcc's OCPP handler is NOT invoked; upstream is the authoritative Central
//     System for these messages.  This ensures the pay backend controls auth,
//     issues its own transaction IDs, and sees consistent Start/Stop pairs.
//
//  2. Sidecar observation for informational messages (BootNotification,
//     StatusNotification, MeterValues, Heartbeat, …):
//       charger → evcc (processed normally by evcc's OCPP handler)
//         → also forwarded to upstream sidecar
//     Upstream observes the session; evcc manages the charger normally.
//
// Upstream → Charger (commands):
//   Calls (type 2) from upstream are injected into the charger via server.Write.
//   The charger's CallResult/CallError is forwarded back to upstream.
//   Examples: RemoteStartTransaction, RemoteStopTransaction, GetConfiguration,
//   ChangeConfiguration, TriggerMessage, SetChargingProfile.
//
// Read-only mode: upstream may observe but cannot send commands; any incoming
// Call from upstream is answered with a SecurityError and not forwarded.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ws"
)

// ForwarderRule maps a station ID (or "*" for all chargers) to an upstream
// OCPP server URL.  Already declared in instance.go.

// package-level forwarder state — protected by forwarderMu
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

// ApplyForwarderRules updates the forwarding rules at runtime.
// Takes effect immediately for new charger connections.
func ApplyForwarderRules(rules []ForwarderRule) {
	forwarderMu.Lock()
	forwarderRules = rules
	forwarderMu.Unlock()
}

// ForwarderRules returns the current forwarding rules.
func ForwarderRules() []ForwarderRule {
	forwarderMu.RLock()
	defer forwarderMu.RUnlock()
	return forwarderRules
}

// StartForwarder is a no-op in the sidecar model; hooks are installed when
// the intercepting server is created and fire on every charger connection.
func StartForwarder() {}

// ── OCPP frame constants ──────────────────────────────────────────────────────

const (
	ocppMsgCall       = 2
	ocppMsgCallResult = 3
	ocppMsgCallError  = 4
)

// actionsRelayedToUpstream lists the OCPP actions for which upstream is the
// authoritative Central System.  For these messages evcc's own OCPP handler is
// bypassed and upstream's CallResult/CallError is relayed back to the charger.
// This is required so that the pay backend controls authorization and issues
// its own transaction IDs.
var actionsRelayedToUpstream = map[string]bool{
	"Authorize":        true,
	"StartTransaction": true,
	"StopTransaction":  true,
	"DataTransfer":     true,
}

var forwarderLog = util.NewLogger("ocpp-forwarder")

// ── interceptingServer ────────────────────────────────────────────────────────

// interceptingServer wraps ws.Server and adds hooks that fire on every new
// connection, every disconnection, and every incoming raw OCPP frame.
// msgHook returns true to bypass evcc's own OCPP message handler (used for
// messages relayed exclusively to upstream).
type interceptingServer struct {
	ws.Server
	mu             sync.RWMutex
	msgHook        func(ws.Channel, []byte) bool
	connectHook    func(ws.Channel)
	disconnectHook func(ws.Channel)
}

// SetMessageHandler wraps the upstream handler to invoke msgHook first.
// If msgHook returns true the evcc OCPP handler is skipped.
func (s *interceptingServer) SetMessageHandler(handler ws.MessageHandler) {
	s.Server.SetMessageHandler(func(ch ws.Channel, data []byte) error {
		s.mu.RLock()
		h := s.msgHook
		s.mu.RUnlock()
		if h != nil {
			if bypass := h(ch, data); bypass {
				return nil // upstream handles this message; evcc must not respond
			}
		}
		return handler(ch, data)
	})
}

// SetNewClientHandler wraps the upstream handler to invoke connectHook first.
func (s *interceptingServer) SetNewClientHandler(handler ws.ConnectedHandler) {
	s.Server.SetNewClientHandler(func(ch ws.Channel) {
		s.mu.RLock()
		h := s.connectHook
		s.mu.RUnlock()
		if h != nil {
			h(ch)
		}
		handler(ch)
	})
}

// SetDisconnectedClientHandler wraps the upstream handler to invoke
// disconnectHook first.
func (s *interceptingServer) SetDisconnectedClientHandler(handler func(ws.Channel)) {
	s.Server.SetDisconnectedClientHandler(func(ch ws.Channel) {
		s.mu.RLock()
		h := s.disconnectHook
		s.mu.RUnlock()
		if h != nil {
			h(ch)
		}
		handler(ch)
	})
}

// newInterceptingServer creates the intercepting ws.Server used by Instance().
func newInterceptingServer() ws.Server {
	s := &interceptingServer{Server: ws.NewServer()}
	s.connectHook = func(ch ws.Channel) { onChargerConnect(s, ch) }
	s.disconnectHook = onChargerDisconnect
	s.msgHook = onChargerMessage
	return s
}

// ── sidecar management ────────────────────────────────────────────────────────

// sidecar holds an upstream connection for a single charger.
type sidecar struct {
	chargerID   string
	upstreamURL string
	conn        *websocket.Conn
	srv         *interceptingServer

	// pendingUpstreamCalls tracks message IDs of Calls sent by upstream to the
	// charger (injected via srv.Write).  When the charger replies, onChargerMessage
	// forwards that reply back to upstream.
	pendingUpstreamCallsMu sync.Mutex
	pendingUpstreamCalls   map[string]struct{}

	// pendingChargerCalls tracks message IDs of charger-initiated Calls for which
	// evcc's handler was bypassed (actionsRelayedToUpstream).  When upstream replies
	// with a CallResult/CallError for one of these IDs, readFromUpstream relays it
	// directly back to the charger so the charger receives the authoritative response.
	pendingChargerCallsMu sync.Mutex
	pendingChargerCalls   map[string]struct{}

	// meterInterval throttles MeterValues forwarded to upstream.
	// When > 0, only one MeterValues frame per interval is forwarded; the rest are
	// silently dropped so upstream receives less traffic while evcc still processes
	// every frame for energy management.
	meterInterval  time.Duration
	lastMeterFwdMu sync.Mutex
	lastMeterFwd   time.Time
}

var (
	sidecarsMu sync.Mutex
	sidecars   = make(map[string]*sidecar)

	// pendingMsgs holds raw frames received from a charger while the upstream
	// sidecar is still being dialled.  Once the sidecar connects the buffered
	// frames are flushed in order, ensuring that BootNotification (and any
	// other early messages) reach the upstream.
	pendingMu   sync.Mutex
	pendingMsgs = make(map[string][][]byte)
)

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

// onChargerConnect fires when a charger establishes a WebSocket connection to
// evcc.  If a matching rule exists, a pending-message buffer is opened and a
// sidecar connection to upstream is dialled in a goroutine.
func onChargerConnect(srv *interceptingServer, ch ws.Channel) {
	id := ch.ID()
	rule, ok := resolveRule(id)
	if !ok {
		return
	}
	// Open the pending buffer before launching the goroutine so that any
	// onChargerMessage calls that arrive before the sidecar is ready are
	// captured rather than dropped.
	pendingMu.Lock()
	pendingMsgs[id] = nil
	pendingMu.Unlock()

	go dialUpstreamSidecar(srv, id, rule)
}

func dialUpstreamSidecar(srv *interceptingServer, id string, rule ForwarderRule) {
	upstreamBase := strings.TrimRight(rule.UpstreamURL, "/")
	upstreamPath := "/" + strings.TrimLeft(id, "/")
	if rule.UpstreamStationID != "" {
		upstreamPath = "/" + strings.TrimLeft(rule.UpstreamStationID, "/")
	}

	var header http.Header
	if rule.Password != "" {
		stationID := strings.TrimLeft(upstreamPath, "/")
		header = authHeader(stationID, rule.Password)
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: rule.Insecure}
	if rule.CaCert != "" {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(rule.CaCert)); !ok {
			forwarderLog.WARN.Printf("forwarder: failed to parse CA cert for %s — forwarding disabled", id)
			drainPendingWithErrors(srv, id, nil)
			return
		}
		tlsConfig.RootCAs = caCertPool
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, _, err := websocket.Dial(ctx, upstreamBase+upstreamPath, &websocket.DialOptions{
		Subprotocols: []string{"ocpp1.6"},
		HTTPHeader:   header,
		HTTPClient:   &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}},
	})
	cancel()
	if err != nil {
		forwarderLog.WARN.Printf("forwarder: dial upstream for %s: %v — forwarding disabled", id, err)
		drainPendingWithErrors(srv, id, nil)
		return
	}
	conn.SetReadLimit(-1) // no limit; OCPP frames can be large

	sc := &sidecar{
		chargerID:            id,
		upstreamURL:          upstreamBase,
		conn:                 conn,
		srv:                  srv,
		pendingUpstreamCalls: make(map[string]struct{}),
		pendingChargerCalls:  make(map[string]struct{}),
	}

	// Atomically install the sidecar and drain the pending buffer.
	sidecarsMu.Lock()
	pendingMu.Lock()
	sidecars[id] = sc
	buffered := pendingMsgs[id]
	delete(pendingMsgs, id)
	pendingMu.Unlock()
	sidecarsMu.Unlock()

	// Forward buffered Calls (type 2) that arrived while we were dialling.
	// For actionsRelayedToUpstream, register the msgID in pendingChargerCalls
	// so that upstream's response will be relayed back to the charger.
	// CallResults/Errors are skipped: evcc already replied to those and
	// upstream has no context for them.
	flushed := 0
	for _, frame := range buffered {
		msgType, msgID, action, err := parseOCPPFrame(frame)
		if err != nil || msgType != ocppMsgCall {
			continue
		}
		if actionsRelayedToUpstream[action] {
			sc.pendingChargerCallsMu.Lock()
			sc.pendingChargerCalls[msgID] = struct{}{}
			sc.pendingChargerCallsMu.Unlock()
		}
		if err := conn.Write(context.Background(), websocket.MessageText, frame); err != nil {
			forwarderLog.ERROR.Printf("forwarder: write buffered frame to upstream for %s: %v", id, err)
			conn.CloseNow()
			notifyUpdated()
			return
		}
		flushed++
	}
	if flushed > 0 {
		forwarderLog.DEBUG.Printf("forwarder: flushed %d call(s) to upstream for %s (skipped %d non-call frame(s))",
			flushed, id, len(buffered)-flushed)
	}

	notifyUpdated()
	forwarderLog.INFO.Printf("forwarder: %s → %s", id, upstreamBase+upstreamPath)

	sc.readFromUpstream(rule.ReadOnly)
}

// drainPendingWithErrors discards the pending buffer for charger id.
// For any buffered Call with an action in actionsRelayedToUpstream, a
// CallError is sent back to the charger so it is not left hanging.
// sc may be nil (used when the sidecar dial itself failed).
func drainPendingWithErrors(srv *interceptingServer, id string, sc *sidecar) {
	pendingMu.Lock()
	buffered := pendingMsgs[id]
	delete(pendingMsgs, id)
	pendingMu.Unlock()

	for _, frame := range buffered {
		msgType, msgID, action, err := parseOCPPFrame(frame)
		if err != nil || msgType != ocppMsgCall || !actionsRelayedToUpstream[action] {
			continue
		}
		errFrame, _ := json.Marshal([]interface{}{
			ocppMsgCallError, msgID, "GenericError",
			"Upstream OCPP server unavailable",
			map[string]any{},
		})
		if writeErr := srv.Write(id, errFrame); writeErr != nil {
			forwarderLog.WARN.Printf("forwarder: send error for pending call %s to %s: %v", msgID, id, writeErr)
		}
	}

	// Also error out any already-tracked pendingChargerCalls (sidecar disconnected mid-session).
	if sc != nil {
		sc.pendingChargerCallsMu.Lock()
		for msgID := range sc.pendingChargerCalls {
			errFrame, _ := json.Marshal([]interface{}{
				ocppMsgCallError, msgID, "GenericError",
				"Upstream OCPP server disconnected",
				map[string]any{},
			})
			if writeErr := srv.Write(sc.chargerID, errFrame); writeErr != nil {
				forwarderLog.WARN.Printf("forwarder: send disconnect error for %s to %s: %v", msgID, sc.chargerID, writeErr)
			}
		}
		sc.pendingChargerCalls = make(map[string]struct{})
		sc.pendingChargerCallsMu.Unlock()
	}
}

// onChargerDisconnect fires when a charger disconnects.  The sidecar upstream
// connection (if any) is closed.
func onChargerDisconnect(ch ws.Channel) {
	id := ch.ID()

	// Discard any pending buffer for this charger.
	pendingMu.Lock()
	delete(pendingMsgs, id)
	pendingMu.Unlock()

	sidecarsMu.Lock()
	sc, ok := sidecars[id]
	if ok {
		delete(sidecars, id)
	}
	sidecarsMu.Unlock()
	if ok {
		sc.conn.CloseNow()
		notifyUpdated()
		forwarderLog.DEBUG.Printf("forwarder: %s upstream connection closed", id)
	}
}

// onChargerMessage fires for every raw OCPP frame received from a charger.
// Returns true if evcc's OCPP handler should be bypassed (upstream is the
// authoritative responder for this message).
//
//   - Call (type 2), action in actionsRelayedToUpstream:
//     forwarded to upstream; evcc handler bypassed; upstream's response is
//     relayed back to charger by readFromUpstream.
//
//   - Call (type 2), other actions:
//     forwarded to upstream AND processed by evcc's handler (sidecar mode).
//
//   - CallResult/CallError (type 3/4):
//     forwarded to upstream only when the msgID matches a Call that upstream
//     initiated; otherwise discarded (evcc already replied to evcc-initiated
//     calls and upstream has no context for them).
func onChargerMessage(ch ws.Channel, data []byte) bool {
	id := ch.ID()

	msgType, msgID, action, err := parseOCPPFrame(data)
	if err != nil {
		return false
	}

	sidecarsMu.Lock()
	sc := sidecars[id]
	sidecarsMu.Unlock()

	switch msgType {
	case ocppMsgCall:
		relay := actionsRelayedToUpstream[action]

		if sc != nil {
			// Throttle MeterValues forwarded to upstream when meterInterval is set.
			// evcc still processes every frame; we only suppress the upstream write.
			if action == "MeterValues" && sc.meterInterval > 0 {
				sc.lastMeterFwdMu.Lock()
				elapsed := time.Since(sc.lastMeterFwd)
				if elapsed < sc.meterInterval {
					sc.lastMeterFwdMu.Unlock()
					return false // evcc handles normally; upstream skipped this time
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

		// Sidecar not yet established — buffer if we have a pending slot.
		pendingMu.Lock()
		_, hasPending := pendingMsgs[id]
		if hasPending {
			pendingMsgs[id] = append(pendingMsgs[id], slices.Clone(data))
		}
		pendingMu.Unlock()

		// Bypass evcc for relay actions even while buffering: the response will
		// come from upstream once the sidecar connects and flushes the buffer.
		return relay && hasPending

	case ocppMsgCallResult, ocppMsgCallError:
		// Forward only if this is the reply to an upstream-initiated Call.
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
		return false // evcc may also process the response (it won't act on unknown IDs)
	}
	return false
}

// readFromUpstream reads frames from the upstream connection.
//
//   - Calls (type 2): injected into the charger via server.Write; msgID tracked
//     in pendingUpstreamCalls so the charger's response is routed back upstream.
//   - CallResults/Errors for pendingChargerCalls: relayed to the charger (these
//     are upstream's authoritative responses to Authorize/StartTransaction/etc.).
//   - All other CallResults/Errors: discarded (evcc already replied to the charger).
//
// In read-only mode, upstream Calls are rejected with a SecurityError.
func (sc *sidecar) readFromUpstream(readOnly bool) {
	defer func() {
		sidecarsMu.Lock()
		if sidecars[sc.chargerID] == sc {
			delete(sidecars, sc.chargerID)
		}
		sidecarsMu.Unlock()
		// Send errors to charger for any pending relayed calls that will never
		// receive a response now that upstream has disconnected.
		drainPendingWithErrors(sc.srv, sc.chargerID, sc)
		sc.conn.CloseNow()
		notifyUpdated()
	}()

	for {
		_, msg, err := sc.conn.Read(context.Background())
		if err != nil {
			forwarderLog.DEBUG.Printf("forwarder: upstream disconnected for %s: %v", sc.chargerID, err)
			return
		}

		msgType, msgID, action, err := parseOCPPFrame(msg)
		if err != nil {
			forwarderLog.WARN.Printf("forwarder: upstream parse error for %s: %v", sc.chargerID, err)
			continue
		}

		switch msgType {
		case ocppMsgCall:
			if readOnly {
				forwarderLog.DEBUG.Printf("forwarder: blocking upstream call %s in read-only session %s", msgID, sc.chargerID)
				errFrame, _ := json.Marshal([]interface{}{
					ocppMsgCallError, msgID, "SecurityError",
					"Charger control not allowed: forwarder is in read-only mode",
					map[string]any{},
				})
				_ = sc.conn.Write(context.Background(), websocket.MessageText, errFrame)
				continue
			}

			// Intercept ChangeConfiguration for MeterValueSampleInterval:
			// absorb the request, apply the interval as our throttle, and
			// reply Accepted to upstream without touching the charger's config.
			if action == "ChangeConfiguration" {
				if interval, ok := extractMeterValueSampleInterval(msg); ok {
					sc.lastMeterFwdMu.Lock()
					sc.meterInterval = interval
					sc.lastMeterFwdMu.Unlock()
					forwarderLog.DEBUG.Printf("forwarder: upstream set MeterValueSampleInterval=%v for %s", interval, sc.chargerID)
					accepted, _ := json.Marshal([]interface{}{
						ocppMsgCallResult, msgID, map[string]any{"status": "Accepted"},
					})
					_ = sc.conn.Write(context.Background(), websocket.MessageText, accepted)
					continue
				}
			}

			// Track the message ID so that onChargerMessage can forward the
			// charger's CallResult/CallError back to upstream.
			sc.pendingUpstreamCallsMu.Lock()
			sc.pendingUpstreamCalls[msgID] = struct{}{}
			sc.pendingUpstreamCallsMu.Unlock()

			// Inject upstream command into the charger.
			if sc.srv != nil {
				if err := sc.srv.Write(sc.chargerID, msg); err != nil {
					forwarderLog.ERROR.Printf("forwarder: inject upstream call into charger %s: %v", sc.chargerID, err)
				}
			}

		case ocppMsgCallResult, ocppMsgCallError:
			// Check whether this is upstream's authoritative response to a
			// charger-initiated Call (e.g. StartTransaction, Authorize).
			sc.pendingChargerCallsMu.Lock()
			_, isChargerCall := sc.pendingChargerCalls[msgID]
			if isChargerCall {
				delete(sc.pendingChargerCalls, msgID)
			}
			sc.pendingChargerCallsMu.Unlock()

			if isChargerCall {
				// Relay upstream's response directly to the charger.
				// evcc's handler was already bypassed for this Call, so the
				// charger is waiting for exactly this response.
				if sc.srv != nil {
					if err := sc.srv.Write(sc.chargerID, msg); err != nil {
						forwarderLog.ERROR.Printf("forwarder: relay upstream response to charger %s: %v", sc.chargerID, err)
					}
				}
				continue
			}
			// Discard: evcc already replied to the charger for non-relay Calls.
		}
	}
}

// ── status & callbacks ────────────────────────────────────────────────────────

// ForwarderSessionStatus holds the observable state of one active forwarder
// session.
type ForwarderSessionStatus struct {
	ChargerID         string `json:"chargerId"`
	UpstreamURL       string `json:"upstreamUrl"`
	UpstreamConnected bool   `json:"upstreamConnected"`
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
	} else {
		forwarderLog.WARN.Printf("forwarder: notifyUpdated called but no callback registered")
	}
}

// SetForwarderUpdated registers a callback invoked whenever a sidecar session
// connects or disconnects.
func SetForwarderUpdated(cb func()) {
	forwarderCbMu.Lock()
	forwarderUpdatedCb = cb
	forwarderCbMu.Unlock()
}

// GetForwarderStatus returns a snapshot of all active forwarder sessions.
func GetForwarderStatus() []ForwarderSessionStatus {
	sidecarsMu.Lock()
	defer sidecarsMu.Unlock()
	out := make([]ForwarderSessionStatus, 0, len(sidecars))
	for _, sc := range sidecars {
		out = append(out, ForwarderSessionStatus{
			ChargerID:         sc.chargerID,
			UpstreamURL:       sc.upstreamURL,
			UpstreamConnected: true,
		})
	}
	return out
}

// ── helpers ───────────────────────────────────────────────────────────────────

// authHeader returns an HTTP Basic Auth header for the given station ID and
// password.
func authHeader(stationID, password string) http.Header {
	creds := base64.StdEncoding.EncodeToString([]byte(stationID + ":" + password))
	h := make(http.Header)
	h.Set("Authorization", "Basic "+creds)
	return h
}

// extractMeterValueSampleInterval parses a ChangeConfiguration OCPP Call and
// returns the duration if the key is MeterValueSampleInterval.
func extractMeterValueSampleInterval(msg []byte) (time.Duration, bool) {
	var frame []json.RawMessage
	if err := json.Unmarshal(msg, &frame); err != nil || len(frame) < 4 {
		return 0, false
	}
	var payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(frame[3], &payload); err != nil {
		return 0, false
	}
	if payload.Key != "MeterValueSampleInterval" {
		return 0, false
	}
	secs, err := strconv.Atoi(payload.Value)
	if err != nil || secs <= 0 {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

// parseOCPPFrame extracts the message type, message-id and (for type-2 Calls)
// the action name from a raw OCPP JSON frame.
func parseOCPPFrame(msg []byte) (msgType int, msgID string, action string, err error) {
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
	if msgType == ocppMsgCall && len(frame) >= 3 {
		_ = json.Unmarshal(frame[2], &action) // best-effort; empty string if missing
	}
	return msgType, msgID, action, nil
}
