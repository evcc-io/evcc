package ocpp

// Read-only mirrors: a ForwarderRule can list further upstreams that only watch.
// Every Call of the charger is copied to them. They never answer the charger,
// never take part in relaying and cannot send commands: Calls from a mirror's
// upstream are answered with a CallError.
//
// Design: docs/agents/ocpp-forwarder.md

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocppj"
)

// mirrorBacklog bounds the frames kept for a mirror that is offline; the oldest are dropped first
const mirrorBacklog = 1000

// MirrorRule is a read-only upstream that receives a copy of a charger's messages.
type MirrorRule struct {
	UpstreamURL       string `json:"upstreamUrl" yaml:"upstreamUrl"`
	UpstreamStationID string `json:"upstreamStationId,omitempty" yaml:"upstreamStationId,omitempty"`
	Username          string `json:"username,omitempty" yaml:"username,omitempty"`
	Password          string `json:"password,omitempty" yaml:"password,omitempty"`
	Insecure          bool   `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	CaCert            string `json:"caCert,omitempty" yaml:"caCert,omitempty"`
}

func (m MirrorRule) Redacted() MirrorRule {
	m.Password = util.Masked(m.Password)
	m.CaCert = util.Masked(m.CaCert)
	return m
}

// Key identifies a mirror within its rule.
func (m MirrorRule) Key() string {
	return strings.TrimRight(m.UpstreamURL, "/") + "|" + m.UpstreamStationID
}

// dialRule is the forwarder rule used to open the mirror's connection for a charger.
func (m MirrorRule) dialRule(chargerID string) ForwarderRule {
	return ForwarderRule{
		StationID:         chargerID,
		UpstreamURL:       m.UpstreamURL,
		UpstreamStationID: m.UpstreamStationID,
		Username:          m.Username,
		Password:          m.Password,
		Insecure:          m.Insecure,
		CaCert:            m.CaCert,
	}
}

// MirrorStatus is the observable state of one mirror.
type MirrorStatus struct {
	UpstreamURL       string `json:"upstreamUrl"`
	UpstreamConnected bool   `json:"upstreamConnected"`
	Error             string `json:"error,omitempty"`
}

type mirrorSession struct {
	chargerID string
	rule      MirrorRule
	out       chan []byte
	stop      chan struct{}
	connected bool // guarded by mirrorsMu
}

var (
	mirrorsMu sync.Mutex

	// running mirrors per charger id, then mirror key
	mirrorSessions = make(map[string]map[string]*mirrorSession)

	// last connection failure per charger id and mirror key
	mirrorErrors = make(map[string]map[string]string)

	// last StatusNotification frame per charger id and connector, replayed on (re)connect
	lastStatus = make(map[string]map[int][]byte)
)

// startMirrors reconciles the running mirrors of a charger with the rule's list:
// new ones are dialled, changed ones redialled and removed ones closed.
func startMirrors(chargerID string, rules []MirrorRule) {
	wanted := make(map[string]MirrorRule, len(rules))
	for _, r := range rules {
		if r.UpstreamURL != "" {
			wanted[r.Key()] = r
		}
	}

	mirrorsMu.Lock()
	running := mirrorSessions[chargerID]
	if running == nil {
		running = make(map[string]*mirrorSession)
		mirrorSessions[chargerID] = running
	}

	var stale []*mirrorSession
	for key, ms := range running {
		if r, ok := wanted[key]; !ok || r != ms.rule {
			stale = append(stale, ms)
			delete(running, key)
		}
	}
	for key := range mirrorErrors[chargerID] {
		if _, ok := wanted[key]; !ok {
			delete(mirrorErrors[chargerID], key)
		}
	}

	var fresh []*mirrorSession
	for key, r := range wanted {
		if _, ok := running[key]; ok {
			continue
		}
		ms := &mirrorSession{
			chargerID: chargerID,
			rule:      r,
			out:       make(chan []byte, mirrorBacklog),
			stop:      make(chan struct{}),
		}
		running[key] = ms
		fresh = append(fresh, ms)
	}
	mirrorsMu.Unlock()

	for _, ms := range stale {
		close(ms.stop)
	}
	for _, ms := range fresh {
		go ms.run()
	}

	notifyUpdated()
}

// stopMirrors closes all mirrors of a charger.
func stopMirrors(chargerID string) {
	mirrorsMu.Lock()
	running := mirrorSessions[chargerID]
	delete(mirrorSessions, chargerID)
	delete(mirrorErrors, chargerID)
	delete(lastStatus, chargerID)
	mirrorsMu.Unlock()

	for _, ms := range running {
		close(ms.stop)
	}

	if len(running) > 0 {
		notifyUpdated()
	}
}

// mirrorCall copies a Call of the charger to its mirrors.
func mirrorCall(chargerID, action string, frame []byte) {
	mirrorsMu.Lock()
	defer mirrorsMu.Unlock()

	if action == "StatusNotification" {
		if connector, ok := statusConnector(frame); ok {
			if lastStatus[chargerID] == nil {
				lastStatus[chargerID] = make(map[int][]byte)
			}
			lastStatus[chargerID][connector] = append([]byte(nil), frame...)
		}
	}

	for _, ms := range mirrorSessions[chargerID] {
		ms.enqueue(append([]byte(nil), frame...))
	}
}

// enqueue queues a frame, dropping the oldest when the backlog is full.
func (ms *mirrorSession) enqueue(frame []byte) {
	for {
		select {
		case ms.out <- frame:
			return
		default:
		}

		select {
		case <-ms.out:
		default:
		}
	}
}

func statusConnector(frame []byte) (int, bool) {
	var msg []json.RawMessage
	if json.Unmarshal(frame, &msg) != nil || len(msg) < 4 {
		return 0, false
	}

	var payload struct {
		ConnectorID int `json:"connectorId"`
	}
	if json.Unmarshal(msg[3], &payload) != nil {
		return 0, false
	}

	return payload.ConnectorID, true
}

// replayFrames are the frames a freshly connected mirror gets first: the charger's boot
// and its last status per connector, with fresh message ids since backends expect a
// boot before anything else.
func (ms *mirrorSession) replayFrames() [][]byte {
	bootMu.Lock()
	boot := lastBoot[ms.chargerID]
	bootMu.Unlock()

	mirrorsMu.Lock()
	statuses := make([][]byte, 0, len(lastStatus[ms.chargerID]))
	for _, frame := range lastStatus[ms.chargerID] {
		statuses = append(statuses, frame)
	}
	mirrorsMu.Unlock()

	var frames [][]byte
	if boot != nil {
		if frame, err := withMessageID(boot, fmt.Sprintf("evcc-mirror-boot-%d", time.Now().UnixNano())); err == nil {
			frames = append(frames, frame)
		}
	}
	for i, status := range statuses {
		if frame, err := withMessageID(status, fmt.Sprintf("evcc-mirror-status-%d-%d", time.Now().UnixNano(), i)); err == nil {
			frames = append(frames, frame)
		}
	}

	return frames
}

func (ms *mirrorSession) stopped() bool {
	select {
	case <-ms.stop:
		return true
	default:
		return false
	}
}

func (ms *mirrorSession) setState(connected bool, err error) {
	if ms.stopped() {
		return
	}

	mirrorsMu.Lock()
	ms.connected = connected

	if mirrorErrors[ms.chargerID] == nil {
		mirrorErrors[ms.chargerID] = make(map[string]string)
	}
	if err != nil {
		mirrorErrors[ms.chargerID][ms.rule.Key()] = err.Error()
	} else {
		delete(mirrorErrors[ms.chargerID], ms.rule.Key())
	}
	mirrorsMu.Unlock()

	notifyUpdated()
}

// run keeps the mirror connected until it is stopped, redialling with backoff.
func (ms *mirrorSession) run() {
	bo := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(reconnectInitialDelay),
		backoff.WithMaxInterval(reconnectMaxDelay),
		backoff.WithMaxElapsedTime(0),
	)

	for !ms.stopped() {
		if ms.serve() {
			bo.Reset()
		}

		select {
		case <-ms.stop:
			return
		case <-time.After(bo.NextBackOff()):
		}
	}
}

// serve runs one connection and reports whether it got connected.
func (ms *mirrorSession) serve() bool {
	conn, err := dialUpstream(ms.chargerID, ms.rule.dialRule(ms.chargerID))
	if err != nil {
		forwarderLog.WARN.Printf("mirror %s: dial %s: %v", ms.chargerID, ms.rule.UpstreamURL, err)
		ms.setState(false, err)
		return false
	}
	conn.SetReadLimit(-1)

	if ms.stopped() {
		conn.CloseNow()
		return false
	}

	forwarderLog.INFO.Printf("mirror %s → %s (read-only)", ms.chargerID, strings.TrimRight(ms.rule.UpstreamURL, "/")+ms.rule.dialRule(ms.chargerID).upstreamPath(ms.chargerID))

	for _, frame := range ms.replayFrames() {
		if err := writeFrame(conn, frame); err != nil {
			conn.CloseNow()
			ms.setState(false, err)
			return true
		}
	}

	ms.setState(true, nil)

	readErr := make(chan error, 1)
	go func() { readErr <- ms.rejectCalls(conn) }()

	for {
		select {
		case <-ms.stop:
			conn.Close(websocket.StatusNormalClosure, "")
			return true

		case err := <-readErr:
			forwarderLog.DEBUG.Printf("mirror %s: upstream disconnected: %v", ms.chargerID, err)
			conn.CloseNow()
			ms.setState(false, err)
			return true

		case frame := <-ms.out:
			if err := writeFrame(conn, frame); err != nil {
				forwarderLog.DEBUG.Printf("mirror %s: write: %v", ms.chargerID, err)
				conn.CloseNow()
				ms.setState(false, err)
				return true
			}
		}
	}
}

func writeFrame(conn *websocket.Conn, frame []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, frame)
}

// rejectCalls reads from the mirror's upstream until it fails. Calls are answered with a
// CallError, results and errors are discarded: the upstream only listens.
func (ms *mirrorSession) rejectCalls(conn *websocket.Conn) error {
	for {
		_, msg, err := conn.Read(context.Background())
		if err != nil {
			return err
		}

		msgType, msgID, _, err := parseOCPPFrame(msg)
		if err != nil || msgType != ocppj.CALL {
			continue
		}

		errFrame, _ := (&ocppj.CallError{
			MessageTypeId:    ocppj.CALL_ERROR,
			UniqueId:         msgID,
			ErrorCode:        ocppj.SecurityError,
			ErrorDescription: "Read-only upstream: commands are not accepted",
		}).MarshalJSON()
		_ = writeFrame(conn, errFrame)
	}
}

// mirrorStatus returns the state of a charger's mirrors in the order of the rule.
func mirrorStatus(chargerID string, rules []MirrorRule) []MirrorStatus {
	mirrorsMu.Lock()
	defer mirrorsMu.Unlock()

	out := make([]MirrorStatus, 0, len(rules))
	for _, r := range rules {
		st := MirrorStatus{UpstreamURL: strings.TrimRight(r.UpstreamURL, "/")}
		if ms, ok := mirrorSessions[chargerID][r.Key()]; ok && ms.connected {
			st.UpstreamConnected = true
		} else if msg, ok := mirrorErrors[chargerID][r.Key()]; ok {
			st.Error = msg
		}
		out = append(out, st)
	}
	return out
}
