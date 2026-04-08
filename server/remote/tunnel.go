package remote

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
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
	cancel        func()
	onStateChange func()

	mu      sync.Mutex
	session *yamux.Session
}

// NewTunnel creates a new tunnel client.
func NewTunnel(tunnelURL, token string, httpHandler http.Handler, log *util.Logger, onStateChange func()) *Tunnel {
	return &Tunnel{
		tunnelURL:     tunnelURL,
		token:         token,
		httpHandler:   httpHandler,
		log:           log,
		onStateChange: onStateChange,
	}
}

// run establishes the tunnel and reconnects on failure.
func (t *Tunnel) run() {
	bo := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Second),
		backoff.WithMaxInterval(60*time.Second),
		backoff.WithMaxElapsedTime(0),
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	for {
		ok, err := t.connect(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.log.ERROR.Printf("tunnel: %v", err)
		}

		// reset backoff after successful connection
		if ok {
			bo.Reset()
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(bo.NextBackOff()):
		}
	}
}

func (t *Tunnel) connect(ctx context.Context) (bool, error) {
	conn, _, err := websocket.Dial(ctx, t.tunnelURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization":   []string{"Bearer " + t.token},
			"X-Sponsor-Token": []string{sponsor.Token},
		},
	})
	if err != nil {
		return false, fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close(websocket.StatusInternalError, "reconnect")

	// wrap websocket as net.Conn for yamux
	netConn := websocket.NetConn(ctx, conn, websocket.MessageBinary)

	config := yamux.DefaultConfig()
	config.LogOutput = t.log.TRACE.Writer()

	session, err := yamux.Client(netConn, config)
	if err != nil {
		conn.CloseNow()
		return false, fmt.Errorf("yamux client: %w", err)
	}

	t.changeState(session, nil)

	// accept streams from the proxy
	for {
		if err := ctx.Err(); err != nil {
			return true, err
		}

		srv := &http.Server{
			Handler: basicAuthMiddleware(t.httpHandler),
		}

		if err := srv.Serve(session); err != nil {
			t.changeState(nil, err)
		}
	}
}

func (t *Tunnel) changeState(session *yamux.Session, err error) {
	t.mu.Lock()
	t.session = session
	t.mu.Unlock()

	if t.onStateChange != nil {
		t.onStateChange()
	}

	if session != nil {
		t.log.INFO.Println("tunnel connected")
	} else {
		if errors.Is(err, context.Canceled) {
			t.log.INFO.Println("tunnel disconnected")
		} else {
			t.log.INFO.Println("tunnel disconnected:", err)
		}
	}
}

// IsConnected returns whether the tunnel is currently connected.
func (t *Tunnel) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.session != nil
}

// Close tears down the tunnel.
func (t *Tunnel) Close() {
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
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
