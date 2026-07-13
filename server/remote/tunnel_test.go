package remote

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/hashicorp/yamux"
	"github.com/stretchr/testify/require"
)

// serveSession upgrades the request to a websocket, wraps it in a yamux server
// session, hands it to the sessions channel and blocks until the session closes.
func serveSession(w http.ResponseWriter, r *http.Request, sessions chan<- *yamux.Session) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	netConn := websocket.NetConn(r.Context(), conn, websocket.MessageBinary)

	session, err := yamux.Server(netConn, yamux.DefaultConfig())
	if err != nil {
		netConn.Close()
		return
	}

	sessions <- session
	<-session.CloseChan() // keep the handler (and websocket) alive until closed
}

// tunnelTestServer accepts websocket connections and hands each yamux server
// session to the sessions channel. Simulates the cloud proxy.
func tunnelTestServer(t *testing.T, sessions chan<- *yamux.Session) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveSession(w, r, sessions)
	}))
}

// requireReachable opens a stream through the server session and issues an
// authenticated request, asserting the client's handler answers.
func requireReachable(t *testing.T, session *yamux.Session) {
	t.Helper()

	stream, err := session.Open()
	require.NoError(t, err)
	defer stream.Close()

	req, err := http.NewRequest(http.MethodGet, "http://tunnel/", nil)
	require.NoError(t, err)
	req.SetBasicAuth("user", "pass")
	require.NoError(t, req.Write(stream))

	resp, err := http.ReadResponse(bufio.NewReader(stream), req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "pong", string(body))
}

// TestTunnelReconnect verifies the client becomes reachable again after the
// server closes the tunnel connection and later accepts a new one.
func TestTunnelReconnect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})
	authenticate := func(user, pass string) bool { return true }

	sessions := make(chan *yamux.Session, 4)
	srv := tunnelTestServer(t, sessions)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	tun := NewTunnel(wsURL, "token", handler, authenticate, nil, util.NewLogger("test"), nil)
	go tun.run()
	defer tun.Close()

	// first connection: client reachable through the tunnel
	session := waitSession(t, sessions)
	requireReachable(t, session)
	require.True(t, tun.IsConnected())

	// server drops the tunnel connection
	session.Close()

	// client reconnects (backoff ~1s) and is reachable again
	session = waitSession(t, sessions)
	requireReachable(t, session)
	require.True(t, tun.IsConnected())
}

// TestTunnelRejectedCredentialsStops verifies the client does not retry when
// the proxy rejects credentials (401/403); a new token requires a restart.
func TestTunnelRejectedCredentialsStops(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	tun := NewTunnel(wsURL, "token", nil, nil, nil, util.NewLogger("test"), nil)
	defer tun.Close()

	done := make(chan struct{})
	go func() { tun.run(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after credential rejection")
	}

	require.Equal(t, int32(1), attempts.Load(), "must not retry after credential rejection")
	require.False(t, tun.IsConnected())
}

// TestTunnelReconnectsAfterTransientError verifies the client keeps retrying
// through a transient proxy failure and connects once the proxy recovers.
func TestTunnelReconnectsAfterTransientError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})
	authenticate := func(user, pass string) bool { return true }

	var attempts atomic.Int32
	sessions := make(chan *yamux.Session, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// first attempt fails transiently, later ones succeed
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		serveSession(w, r, sessions)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	tun := NewTunnel(wsURL, "token", handler, authenticate, nil, util.NewLogger("test"), nil)
	go tun.run()
	defer tun.Close()

	session := waitSession(t, sessions)
	requireReachable(t, session)
	require.True(t, tun.IsConnected())
	require.GreaterOrEqual(t, attempts.Load(), int32(2), "must retry after transient failure")
}

func waitSession(t *testing.T, sessions <-chan *yamux.Session) *yamux.Session {
	t.Helper()
	select {
	case s := <-sessions:
		return s
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for tunnel connection")
		return nil
	}
}
