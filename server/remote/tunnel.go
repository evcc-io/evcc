package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/hashicorp/yamux"
)

// Tunnel manages a WebSocket+yamux tunnel to the cloud proxy.
type Tunnel struct {
	tunnelURL     string
	token         string
	httpHandler   http.Handler
	authenticate  func(user, pass string) bool
	trackActivity func(username string, active bool)
	log           *util.Logger
	cancel        func()
	onStateChange func()
	rateLimiter   *authRateLimiter

	mu      sync.Mutex
	session *yamux.Session
}

// NewTunnel creates a new tunnel client.
func NewTunnel(tunnelURL, token string, httpHandler http.Handler, authenticate func(user, pass string) bool, trackActivity func(string, bool), log *util.Logger, onStateChange func()) *Tunnel {
	return &Tunnel{
		tunnelURL:     tunnelURL,
		token:         token,
		httpHandler:   httpHandler,
		authenticate:  authenticate,
		trackActivity: trackActivity,
		log:           log,
		onStateChange: onStateChange,
		rateLimiter:   newAuthRateLimiter(),
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

	netConn := websocket.NetConn(ctx, conn, websocket.MessageBinary)

	config := yamux.DefaultConfig()
	config.LogOutput = t.log.TRACE.Writer()

	session, err := yamux.Client(netConn, config)
	if err != nil {
		netConn.Close() // closes the underlying socket connection
		return false, fmt.Errorf("yamux client: %w", err)
	}

	t.changeState(session, nil)

	// accept streams from the proxy
	srv := &http.Server{
		Handler: t.basicAuthMiddleware(t.httpHandler),
	}

	if err := srv.Serve(session); err != nil {
		t.changeState(nil, err)
	}

	return true, nil
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
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
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

// LoginBlocked returns whether login attempts are currently blocked by the rate limiter.
func (t *Tunnel) LoginBlocked() bool {
	return !t.rateLimiter.allow()
}

// Close tears down the tunnel.
func (t *Tunnel) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// close websocket; produces io.EOF in yamux which it handles silently
	if t.session != nil {
		t.session.Close() // closes the underlying socket connection
	}

	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
}

// basicAuthMiddleware wraps a handler with HTTP basic auth, validating
// credentials against the given authenticate function per request.
// It rate-limits failed attempts to prevent brute-force attacks.
func (t *Tunnel) basicAuthMiddleware(next http.Handler) http.Handler {
	rejectAuth := func(w http.ResponseWriter) {
		w.Header().Set("WWW-Authenticate", `Basic realm="evcc"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || t.authenticate == nil {
			rejectAuth(w)
			return
		}

		if !t.rateLimiter.allow() {
			t.log.INFO.Printf("login blocked for %q (rate limited)", user)
			http.Error(w, "Too many failed login attempts. Try again in 1 minute.", http.StatusTooManyRequests)
			return
		}

		if !t.authenticate(user, pass) {
			t.rateLimiter.fail()
			t.log.INFO.Printf("failed login attempt for %q", user)
			rejectAuth(w)
			return
		}

		if t.trackActivity != nil {
			t.trackActivity(user, true)
			defer t.trackActivity(user, false) // long-running requests (ws)
		}

		next.ServeHTTP(w, r)
	})
}
