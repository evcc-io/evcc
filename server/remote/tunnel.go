package remote

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/hashicorp/yamux"
)

const (
	basicAuthUser = "admin"
	basicAuthPass = "secret"
)

// Tunnel manages a WebSocket+yamux tunnel to the cloud proxy.
type Tunnel struct {
	tunnelURL     string
	token         string
	httpHandler   http.Handler
	log           *util.Logger
	onStateChange func()

	mu        sync.Mutex
	session   *yamux.Session
	connected bool
	done      chan struct{}
	closeOnce sync.Once
}

// NewTunnel creates a new tunnel client.
func NewTunnel(tunnelURL, token string, httpHandler http.Handler, log *util.Logger, onStateChange func()) *Tunnel {
	return &Tunnel{
		tunnelURL:     tunnelURL,
		token:         token,
		httpHandler:   httpHandler,
		log:           log,
		onStateChange: onStateChange,
		done:          make(chan struct{}),
	}
}

// Connect establishes the tunnel and reconnects on failure.
func (t *Tunnel) Connect() {
	backoff := time.Second

	for {
		select {
		case <-t.done:
			return
		default:
		}

		connected, err := t.dial()
		if err != nil {
			t.log.ERROR.Printf("tunnel: %v", err)
		}

		// reset backoff after successful connection
		if connected {
			backoff = time.Second
		}

		select {
		case <-t.done:
			return
		case <-time.After(backoff):
		}

		if backoff < 60*time.Second {
			backoff *= 2
		}
	}
}

func (t *Tunnel) dial() (bool, error) {
	ctx := context.Background()

	conn, _, err := websocket.Dial(ctx, t.tunnelURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": []string{"Bearer " + t.token},
		},
	})
	if err != nil {
		return false, fmt.Errorf("websocket dial: %w", err)
	}

	// wrap websocket as net.Conn for yamux
	netConn := websocket.NetConn(ctx, conn, websocket.MessageBinary)

	session, err := yamux.Client(netConn, nil)
	if err != nil {
		conn.CloseNow()
		return false, fmt.Errorf("yamux client: %w", err)
	}

	t.mu.Lock()
	t.session = session
	t.connected = true
	t.mu.Unlock()

	t.log.INFO.Println("tunnel connected")

	if t.onStateChange != nil {
		t.onStateChange()
	}

	// accept streams from the proxy
	for {
		stream, err := session.Accept()
		if err != nil {
			t.mu.Lock()
			t.connected = false
			t.session = nil
			t.mu.Unlock()

			t.log.DEBUG.Printf("tunnel session ended: %v", err)

			if t.onStateChange != nil {
				t.onStateChange()
			}

			return true, err
		}

		go t.handleStream(stream)
	}
}

func (t *Tunnel) handleStream(conn net.Conn) {
	defer conn.Close()

	srv := &http.Server{
		Handler: basicAuthMiddleware(t.httpHandler),
	}

	// serve a single HTTP request on this stream
	listener := &singleConnListener{conn: conn}
	_ = srv.Serve(listener)
}

// IsConnected returns whether the tunnel is currently connected.
func (t *Tunnel) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected
}

// Close tears down the tunnel.
func (t *Tunnel) Close() {
	t.closeOnce.Do(func() { close(t.done) })

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.session != nil {
		t.session.Close()
		t.session = nil
	}
	t.connected = false
}

// basicAuthMiddleware wraps a handler with HTTP basic auth.
func basicAuthMiddleware(next http.Handler) http.Handler {
	expected := "Basic " + base64.StdEncoding.EncodeToString(
		[]byte(basicAuthUser+":"+basicAuthPass),
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="evcc"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// singleConnListener implements net.Listener for a single connection.
type singleConnListener struct {
	conn net.Conn
	once sync.Once
	done chan struct{}
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	var conn net.Conn
	l.once.Do(func() {
		conn = l.conn
		l.done = make(chan struct{})
	})
	if conn != nil {
		return conn, nil
	}
	// block until closed — signal http.Server to stop
	<-l.done
	return nil, fmt.Errorf("listener closed")
}

func (l *singleConnListener) Close() error {
	if l.done != nil {
		select {
		case <-l.done:
		default:
			close(l.done)
		}
	}
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
