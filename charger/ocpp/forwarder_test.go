package ocpp

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBootFrame = `[2,"orig-id","BootNotification",{"chargePointVendor":"v","chargePointModel":"m"}]`

func TestForwarderWithMessageID(t *testing.T) {
	frame, err := withMessageID([]byte(testBootFrame), "new-id")
	require.NoError(t, err)

	msgType, msgID, action, err := parseOCPPFrame(frame)
	require.NoError(t, err)
	assert.Equal(t, ocppj.CALL, msgType)
	assert.Equal(t, "new-id", msgID)
	assert.Equal(t, "BootNotification", action)
	assert.Contains(t, string(frame), `"chargePointVendor":"v"`)

	_, err = withMessageID([]byte(`{}`), "new-id")
	assert.Error(t, err)
}

func TestForwarderIdTagStatusFromResult(t *testing.T) {
	for _, tc := range []struct {
		name   string
		frame  string
		status string
		ok     bool
	}{
		{"StartTransaction invalid", `[3,"1",{"idTagInfo":{"status":"Invalid"},"transactionId":42}]`, "Invalid", true},
		{"StartTransaction accepted", `[3,"1",{"idTagInfo":{"status":"Accepted"},"transactionId":42}]`, "Accepted", true},
		{"Authorize blocked", `[3,"1",{"idTagInfo":{"status":"Blocked"}}]`, "Blocked", true},
		{"no idTagInfo", `[3,"1",{"currentTime":"2026-07-30T00:00:00Z"}]`, "", false},
		{"call not result", `[2,"1","StartTransaction",{"idTag":"evcc"}]`, "", false},
		{"malformed", `not json`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, ok := idTagStatusFromResult([]byte(tc.frame))
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.status, string(status))
		})
	}
}

func TestForwarderRejectedAuthorization(t *testing.T) {
	const invalid = `[3,"1",{"idTagInfo":{"status":"Invalid"}}]`

	for _, tc := range []struct {
		name   string
		action string
		frame  string
		status string
		ok     bool
	}{
		{"StartTransaction rejected", "StartTransaction", invalid, "Invalid", true},
		{"Authorize rejected", "Authorize", `[3,"1",{"idTagInfo":{"status":"Blocked"}}]`, "Blocked", true},
		{"StartTransaction accepted", "StartTransaction", `[3,"1",{"idTagInfo":{"status":"Accepted"},"transactionId":42}]`, "", false},
		// StopTransaction.conf may legally carry idTagInfo - not a failed start
		{"StopTransaction rejected", "StopTransaction", invalid, "", false},
		{"DataTransfer", "DataTransfer", invalid, "", false},
		{"relay action unknown", "", invalid, "", false},
		{"no idTagInfo", "StartTransaction", `[3,"1",{}]`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, ok := rejectedAuthorization(tc.action, []byte(tc.frame))
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.status, string(status))
		})
	}
}

// newBufferedCS returns a CS whose TRACE output is captured in buf.
func newBufferedCS(buf *bytes.Buffer) *CS {
	np := jww.NewNotepad(jww.LevelTrace, jww.LevelTrace, buf, io.Discard, "ocpp", 0)
	return &CS{log: &util.Logger{Notepad: np}}
}

func TestForwarderTraceLogging(t *testing.T) {
	const id = "CP01"
	frame := []byte(`[2,"1","StartTransaction",{}]`)

	// received frame (e.g. a bypassed relay Call)
	var recv bytes.Buffer
	newBufferedCS(&recv).traceRecv(id, frame)
	assert.Contains(t, recv.String(), "recv CP01: "+string(frame))

	// frame proxied from upstream is clearly marked
	var up bytes.Buffer
	newBufferedCS(&up).traceSend(id, "upstream", frame)
	assert.Contains(t, up.String(), "send CP01 (upstream): "+string(frame))

	// evcc-generated frame carries no origin annotation
	var evcc bytes.Buffer
	newBufferedCS(&evcc).traceSend(id, "", frame)
	assert.Contains(t, evcc.String(), "send CP01: "+string(frame))
	assert.NotContains(t, evcc.String(), "(upstream)")
}

// fakeChannel implements ws.Channel for hook tests
type fakeChannel string

func (c fakeChannel) ID() string                               { return string(c) }
func (c fakeChannel) RemoteAddr() net.Addr                     { return nil }
func (c fakeChannel) TLSConnectionState() *tls.ConnectionState { return nil }
func (c fakeChannel) IsConnected() bool                        { return true }

func TestForwarderBootNotificationCache(t *testing.T) {
	const id = "fwd-boot-cache-test"

	onChargerMessage(fakeChannel(id), []byte(testBootFrame))

	bootMu.Lock()
	boot := lastBoot[id]
	bootMu.Unlock()
	assert.Equal(t, testBootFrame, string(boot))

	onChargerDisconnect(fakeChannel(id))

	bootMu.Lock()
	boot = lastBoot[id]
	bootMu.Unlock()
	assert.Nil(t, boot)
}

// forwarderTestSetup marks a charger as connected with a cached boot frame and
// fast reconnect, returning a cleanup function.
func forwarderTestSetup(t *testing.T, id string) func() {
	t.Helper()

	delay := reconnectInitialDelay
	reconnectInitialDelay = 10 * time.Millisecond

	sidecarsMu.Lock()
	connectedChargers[id] = true
	sidecarsMu.Unlock()

	bootMu.Lock()
	lastBoot[id] = []byte(testBootFrame)
	bootMu.Unlock()

	return func() {
		ApplyForwarderRules(nil)
		sidecarsMu.Lock()
		delete(connectedChargers, id)
		sidecarsMu.Unlock()
		bootMu.Lock()
		delete(lastBoot, id)
		bootMu.Unlock()
		reconnectInitialDelay = delay
	}
}

// expectBootReplay asserts that a replayed BootNotification with a fresh message id arrives.
func expectBootReplay(t *testing.T, frames <-chan []byte) {
	t.Helper()

	select {
	case data := <-frames:
		msgType, msgID, action, err := parseOCPPFrame(data)
		require.NoError(t, err)
		assert.Equal(t, ocppj.CALL, msgType)
		assert.Equal(t, "BootNotification", action)
		assert.NotEqual(t, "orig-id", msgID)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for upstream reconnect")
	}
}

func TestForwarderUpstreamReconnect(t *testing.T) {
	const id = "fwd-reconnect-test"

	var conns atomic.Int32
	frames := make(chan []byte, 1)

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp1.6"}})
		if err != nil {
			return
		}
		if conns.Add(1) == 1 {
			// simulate upstream dropping the connection without close handshake
			c.CloseNow()
			return
		}
		if _, data, err := c.Read(r.Context()); err == nil {
			select {
			case frames <- data:
			default:
			}
		}
		c.CloseNow()
	}))
	srv.Start()

	cleanup := forwarderTestSetup(t, id)
	defer cleanup()

	ApplyForwarderRules([]ForwarderRule{{StationID: id, UpstreamURL: "ws" + strings.TrimPrefix(srv.URL, "http")}})

	expectBootReplay(t, frames)
	assert.GreaterOrEqual(t, conns.Load(), int32(2))
}

func TestForwarderDialRetry(t *testing.T) {
	const id = "fwd-dial-retry-test"

	var reqs atomic.Int32
	frames := make(chan []byte, 1)

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqs.Add(1) <= 2 {
			// reject the websocket upgrade to simulate an unreachable upstream
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp1.6"}})
		if err != nil {
			return
		}
		if _, data, err := c.Read(r.Context()); err == nil {
			select {
			case frames <- data:
			default:
			}
		}
		c.CloseNow()
	}))
	srv.Start()

	cleanup := forwarderTestSetup(t, id)
	defer cleanup()

	ApplyForwarderRules([]ForwarderRule{{StationID: id, UpstreamURL: "ws" + strings.TrimPrefix(srv.URL, "http")}})

	expectBootReplay(t, frames)
	assert.GreaterOrEqual(t, reqs.Load(), int32(3))
}
