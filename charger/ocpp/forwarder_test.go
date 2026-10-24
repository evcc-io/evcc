package ocpp

import (
	"crypto/tls"
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

func TestForwarderTrackTransaction(t *testing.T) {
	const id = "fwd-track-txn-test"

	cp := NewChargePoint(util.NewLogger("foo"), instance, id)
	cp.connected = true
	conn, err := NewConnector(t.Context(), util.NewLogger("foo"), 1, cp, "", Timeout)
	require.NoError(t, err)

	reg := newRegistration()
	reg.cp = cp
	instance.mu.Lock()
	instance.regs[id] = reg
	instance.mu.Unlock()
	defer func() {
		instance.mu.Lock()
		delete(instance.regs, id)
		instance.mu.Unlock()
	}()

	// rejected start is not tracked
	trackTransaction(instance, id,
		[]byte(`[2,"0","StartTransaction",{"connectorId":1,"idTag":"BLOCKED","meterStart":0,"timestamp":"2026-10-20T17:38:50Z"}]`),
		[]byte(`[3,"0",{"idTagInfo":{"status":"Blocked"},"transactionId":5}]`))
	assert.Empty(t, conn.IdTag())

	// upstream issues the transaction id, the charger's request carries the idTag
	trackTransaction(instance, id,
		[]byte(`[2,"1","StartTransaction",{"connectorId":1,"idTag":"04373B1A7F6F80","meterStart":0,"timestamp":"2026-10-20T17:39:50Z"}]`),
		[]byte(`[3,"1",{"idTagInfo":{"status":"Accepted"},"transactionId":6}]`))

	txn, err := conn.TransactionID()
	require.NoError(t, err)
	assert.Equal(t, 6, txn)
	assert.Equal(t, "04373B1A7F6F80", conn.IdTag())

	trackTransaction(instance, id,
		[]byte(`[2,"2","StopTransaction",{"transactionId":6,"meterStop":1000,"timestamp":"2026-10-20T18:39:50Z"}]`),
		[]byte(`[3,"2",{}]`))

	txn, err = conn.TransactionID()
	require.NoError(t, err)
	assert.Zero(t, txn)
	assert.Empty(t, conn.IdTag())
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
