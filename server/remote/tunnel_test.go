package remote

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/hashicorp/yamux"
	"github.com/stretchr/testify/require"
)

// tunnelTestServer accepts websocket connections, wraps each in a yamux server
// session, and hands them to the sessions channel. Simulates the cloud proxy.
func tunnelTestServer(t *testing.T, sessions chan<- *yamux.Session) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
