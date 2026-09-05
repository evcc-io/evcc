package charger

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openevse"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// full status as sent by the firmware on websocket connect (subset of keys, plus noise keys)
const openevseFullStatus = `{"mode":"STA","amp":16000,"voltage":230,"power":11040,"pilot":16,"state":3,"vehicle":1,"status":"active","elapsed":120,"session_energy":1500,"total_energy":42.5,"temp":false,"manual_override":0}`

// openevseTestServer fakes the firmware's /ws and /claims/{id} endpoints.
// It never answers {"ping":1}, so with shortened timeouts it also models a
// silent (half-open) connection.
type openevseTestServer struct {
	srv  *httptest.Server
	send chan string   // frames pushed to the current websocket client
	drop chan struct{} // closes the current websocket connection

	mu        sync.Mutex
	connects  int
	wsAuth    string
	claimAuth string
	claimPath string
	claims    []openevse.Claim
	deleted   int
	claimErr  bool // reject claim writes with 400
}

// newOpenEVSETestServer starts the fake. If full is empty the websocket never sends a frame.
func newOpenEVSETestServer(t *testing.T, full string) *openevseTestServer {
	t.Helper()

	s := &openevseTestServer{
		send: make(chan string),
		drop: make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.connects++
		s.wsAuth = r.Header.Get("Authorization")
		s.mu.Unlock()

		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")

		ctx := r.Context()
		if full != "" {
			if err := c.Write(ctx, websocket.MessageText, []byte(full)); err != nil {
				return
			}
		}

		for {
			select {
			case msg := <-s.send:
				if err := c.Write(ctx, websocket.MessageText, []byte(msg)); err != nil {
					return
				}
			case <-s.drop:
				return
			case <-ctx.Done():
				return
			}
		}
	})

	mux.HandleFunc("/claims/", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.claimAuth = r.Header.Get("Authorization")
		s.claimPath = r.URL.Path

		if s.claimErr {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"msg":"Could not make claim"}`))
			return
		}

		switch r.Method {
		case http.MethodPost:
			var claim openevse.Claim
			if err := json.NewDecoder(r.Body).Decode(&claim); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			s.claims = append(s.claims, claim)
			_, _ = w.Write([]byte(`{"msg":"done"}`))
		case http.MethodDelete:
			s.deleted++
			_, _ = w.Write([]byte(`{"msg":"done"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)

	return s
}

// failClaims makes every /claims request answer 400
func (s *openevseTestServer) failClaims() {
	s.mu.Lock()
	s.claimErr = true
	s.mu.Unlock()
}

// openevseSnapshot is a lock-free copy of the recorded server state
type openevseSnapshot struct {
	connects  int
	wsAuth    string
	claimAuth string
	claimPath string
	claims    []openevse.Claim
	deleted   int
}

func (s *openevseTestServer) snapshot() openevseSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return openevseSnapshot{
		connects: s.connects, wsAuth: s.wsAuth, claimAuth: s.claimAuth,
		claimPath: s.claimPath, claims: append([]openevse.Claim(nil), s.claims...), deleted: s.deleted,
	}
}

// shortenOpenEVSEReady shortens the constructor's wait for the first frame
func shortenOpenEVSEReady(t *testing.T, d time.Duration) {
	t.Helper()

	prev := openevseReadyTimeout
	openevseReadyTimeout = d
	t.Cleanup(func() { openevseReadyTimeout = prev })
}

// shortenOpenEVSEKeepalive shortens the ping interval and the per-read deadline.
// Both are snapshotted by the constructor, so this must be called before the
// charger under test is created.
func shortenOpenEVSEKeepalive(t *testing.T, ping, read time.Duration) {
	t.Helper()

	prevPing, prevRead := openevsePingInterval, openevseReadTimeout
	openevsePingInterval, openevseReadTimeout = ping, read

	t.Cleanup(func() { openevsePingInterval, openevseReadTimeout = prevPing, prevRead })
}

func newTestOpenEVSE(ctx context.Context, t *testing.T, uri, user, password string) *OpenEVSE {
	t.Helper()
	c, err := NewOpenEVSE(ctx, uri, user, password)
	require.NoError(t, err)
	return c.(*OpenEVSE)
}

func waitOpenEVSEConnected(t *testing.T, c *OpenEVSE) {
	t.Helper()
	require.Eventually(t, func() bool {
		_, err := c.Enabled()
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)
}

// TestOpenEVSEConstructWithoutFrame covers Finding F2 (final round): a device that
// accepts the websocket but never sends a frame - a wrong host answering on the
// port, or firmware without the push channel - must fail construction so evcc's
// device test does not falsely pass.
func TestOpenEVSEConstructWithoutFrame(t *testing.T) {
	s := newOpenEVSETestServer(t, "") // websocket accepts but never sends
	shortenOpenEVSEReady(t, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	_, err := NewOpenEVSE(ctx, s.srv.URL, "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, api.ErrTimeout)
}

// TestOpenEVSEConstructUnauthorized: a 401 at the websocket handshake (wrong
// password) must surface as a constructor error.
func TestOpenEVSEConstructUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	shortenOpenEVSEReady(t, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	_, err := NewOpenEVSE(ctx, srv.URL, "admin", "wrong")
	require.Error(t, err)
	assert.ErrorContains(t, err, "401")
}

// TestOpenEVSEStaleConnection covers Finding F1 (final round): a connection that
// stops delivering frames (half-open TCP) must not leave the charger reporting
// stale state. The read deadline fires, the charger goes unreadable and reconnects.
func TestOpenEVSEStaleConnection(t *testing.T) {
	shortenOpenEVSEKeepalive(t, 100*time.Millisecond, 300*time.Millisecond)

	// the fake never answers {"ping":1}, so the link is silent after the full status
	s := newOpenEVSETestServer(t, openevseFullStatus)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	require.Eventually(t, func() bool {
		_, err := c.Enabled()
		return err != nil
	}, 5*time.Second, 10*time.Millisecond, "stale connection must make reads fail")

	require.Eventually(t, func() bool {
		return s.snapshot().connects >= 2
	}, 5*time.Second, 10*time.Millisecond, "driver must reconnect after the read deadline")
}

func TestOpenEVSEStatusMerge(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	status, err := c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	enabled, err := c.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	power, err := c.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 11040.0, power)

	charged, err := c.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 1.5, charged)

	total, err := c.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 42.5, total)

	dur, err := c.ChargeDuration()
	require.NoError(t, err)
	assert.Equal(t, 120*time.Second, dur)

	cur, err := c.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 16.0, cur)

	// partial update: only changed keys, everything else must survive
	s.send <- `{"state":2,"status":"disabled","power":0,"pilot":0}`

	require.Eventually(t, func() bool {
		status, _ := c.Status()
		return status == api.StatusB
	}, 5*time.Second, 10*time.Millisecond)

	enabled, err = c.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	power, err = c.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 0.0, power)

	total, err = c.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 42.5, total, "unchanged key must keep previous value")

	// vehicle unplugged while sleeping -> A
	s.send <- `{"state":254,"vehicle":0}`
	require.Eventually(t, func() bool {
		status, _ := c.Status()
		return status == api.StatusA
	}, 5*time.Second, 10*time.Millisecond)

	// the pong and other unknown keys are ignored
	s.send <- `{"pong":1}`
	s.send <- `{"free_heap":12345}`
	status, err = c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusA, status)
}

func TestOpenEVSEClaims(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	require.NoError(t, c.MaxCurrent(16))
	require.NoError(t, c.Enable(true))
	require.NoError(t, c.Enable(false))

	snap := s.snapshot()
	assert.Equal(t, "/claims/262145", snap.claimPath)
	assert.Equal(t, []openevse.Claim{
		{State: openevse.Disabled, ChargeCurrent: 16},
		{State: openevse.Enabled, ChargeCurrent: 16},
		{State: openevse.Disabled, ChargeCurrent: 16},
	}, snap.claims)
}

// TestOpenEVSEManualOverrideWarning: a claim written while the firmware's manual
// override (priority 1000) is active must warn once, not on every claim write.
func TestOpenEVSEManualOverrideWarning(t *testing.T) {
	s := newOpenEVSETestServer(t, `{"state":3,"vehicle":1,"status":"active","manual_override":1}`)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	var warnings int
	c.warnOverride = func() { warnings++ }

	require.NoError(t, c.Enable(true))
	assert.Equal(t, 1, warnings, "manual override must warn once")

	require.NoError(t, c.Enable(false))
	assert.Equal(t, 1, warnings, "must not warn again while the override stays active")
}

// TestOpenEVSEManualOverrideWarningRearms: the warning must fire again once the
// override has been observed inactive in between.
func TestOpenEVSEManualOverrideWarningRearms(t *testing.T) {
	s := newOpenEVSETestServer(t, `{"state":3,"vehicle":1,"status":"active","manual_override":1}`)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	var warnings int
	c.warnOverride = func() { warnings++ }

	require.NoError(t, c.Enable(true))
	assert.Equal(t, 1, warnings)

	s.send <- `{"manual_override":0}`
	require.Eventually(t, func() bool {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.status.ManualOverride == 0
	}, 5*time.Second, 10*time.Millisecond)

	s.send <- `{"manual_override":1}`
	require.Eventually(t, func() bool {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.status.ManualOverride == 1
	}, 5*time.Second, 10*time.Millisecond)

	require.NoError(t, c.Enable(false))
	assert.Equal(t, 2, warnings, "must warn again after the override was observed inactive")
}

func TestOpenEVSEClaimError(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	s.failClaims()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")
	assert.Error(t, c.Enable(true))
}

func TestOpenEVSEReconnectReassertsClaim(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	require.NoError(t, c.MaxCurrent(10))
	require.NoError(t, c.Enable(true))
	require.Len(t, s.snapshot().claims, 2)

	// firmware reboots: connection drops, claims are gone
	s.drop <- struct{}{}

	require.Eventually(t, func() bool {
		return s.snapshot().connects == 2
	}, 10*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return len(s.snapshot().claims) == 3
	}, 5*time.Second, 10*time.Millisecond)

	claims := s.snapshot().claims
	assert.Equal(t, openevse.Claim{State: openevse.Enabled, ChargeCurrent: 10}, claims[2])
}

func TestOpenEVSEReconnectWithoutClaim(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	s.drop <- struct{}{}

	require.Eventually(t, func() bool {
		return s.snapshot().connects == 2
	}, 10*time.Second, 10*time.Millisecond)
	waitOpenEVSEConnected(t, c)

	assert.Empty(t, s.snapshot().claims, "no claim must be posted if evcc never wrote one")
}

func TestOpenEVSEReleaseOnShutdown(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")
	require.NoError(t, c.Enable(true))

	cancel()

	require.Eventually(t, func() bool {
		return s.snapshot().deleted == 1
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, "/claims/262145", s.snapshot().claimPath)
}

// TestOpenEVSEReleaseDuringBackoff covers Finding F2 (fix round 1): if ctx is
// cancelled while the driver is stuck in the dial-failure backoff, the claim must
// still be released on shutdown. The websocket connects once so that construction
// succeeds and a claim can be written, then every further dial fails.
func TestOpenEVSEReleaseDuringBackoff(t *testing.T) {
	var mu sync.Mutex
	var attempts, deleted int
	var fail bool

	drop := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		failing := fail
		mu.Unlock()

		if failing {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		if err := conn.Write(r.Context(), websocket.MessageText, []byte(openevseFullStatus)); err != nil {
			return
		}

		select {
		case <-drop:
		case <-r.Context().Done():
		}
	})
	mux.HandleFunc("/claims/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"msg":"done"}`))
		case http.MethodDelete:
			mu.Lock()
			deleted++
			mu.Unlock()
			_, _ = w.Write([]byte(`{"msg":"done"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())

	c := newTestOpenEVSE(ctx, t, srv.URL, "", "")
	require.NoError(t, c.Enable(true))

	// the device goes away: the connection drops and every reconnect fails
	mu.Lock()
	fail = true
	mu.Unlock()
	close(drop)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return attempts >= 3
	}, 10*time.Second, 10*time.Millisecond, "driver must be looping on dial failures")

	// cancel while the driver is in the dial-failure backoff
	cancel()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return deleted == 1
	}, 5*time.Second, 10*time.Millisecond)
}

// TestOpenEVSEReleaseViaShutdownHook covers Fix round 2: evcc never cancels a
// device's context on shutdown (it only cancels on failure), so the deferred
// release() in run() never fires from a real SIGINT/SIGTERM. release() must also
// be reachable through evcc's cmd/shutdown hook registry. shutdown.Cleanup runs
// every hook registered by every OpenEVSE constructed so far in this test binary;
// those from other tests are harmless no-ops (nil claim, or a WARN against an
// already-closed server), so asserting on this test's own server is safe.
func TestOpenEVSEReleaseViaShutdownHook(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")
	require.NoError(t, c.Enable(true))

	doneC := make(chan struct{})
	shutdown.Cleanup(doneC)
	<-doneC

	assert.Equal(t, 1, s.snapshot().deleted)
}

func TestOpenEVSEBasicAuth(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "admin", "secret")
	require.NoError(t, c.Enable(true))

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	snap := s.snapshot()
	assert.Equal(t, want, snap.wsAuth, "websocket handshake must carry basic auth")
	assert.Equal(t, want, snap.claimAuth, "claim request must carry basic auth")
}

func TestOpenEVSEInvalidState(t *testing.T) {
	tc := []struct {
		state   int
		want    api.ChargeStatus // StatusNone means an error is expected
		wantErr string           // substring expected in the error, when want == StatusNone
	}{
		{0, api.StatusNone, "invalid status: 0"}, // unknown
		{4, api.StatusB, ""},                     // vent required, vehicle connected -> B
		{6, api.StatusNone, "gfci fault"},        // gfci fault
		{8, api.StatusNone, "stuck relay"},       // stuck relay
		{11, api.StatusNone, "over current"},     // over current
	}

	for _, tt := range tc {
		s := newOpenEVSETestServer(t, fmt.Sprintf(`{"state":%d,"vehicle":1,"status":"active"}`, tt.state))
		ctx, cancel := context.WithCancel(context.Background())

		c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

		status, err := c.Status()
		if tt.want == api.StatusNone {
			assert.ErrorContains(t, err, tt.wantErr)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.want, status)
		}

		cancel()
	}
}

// TestOpenEVSEGarbageOnlyFrame covers Finding 1 (fix round 1): a first frame that
// fails to decode must not mark the connection ready with a zero Status - with the
// readiness wait in place, construction fails.
func TestOpenEVSEGarbageOnlyFrame(t *testing.T) {
	s := newOpenEVSETestServer(t, "garbage") // not valid JSON
	shortenOpenEVSEReady(t, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	_, err := NewOpenEVSE(ctx, s.srv.URL, "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, api.ErrTimeout)
}

// TestOpenEVSEGarbageFirstFrame: once a later frame decodes (fully or with only an
// UnmarshalTypeError) the charger becomes readable.
func TestOpenEVSEGarbageFirstFrame(t *testing.T) {
	s := newOpenEVSETestServer(t, "garbage") // not valid JSON
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// send is unbuffered and the constructor now blocks until the first usable
	// frame, so the valid frame has to be pushed from another goroutine
	go func() { s.send <- openevseFullStatus }()

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	status, err := c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)
}

// TestOpenEVSEIdentify covers the RFID tag identifier: absent on the full status
// (RFID disabled or no card yet), populated by a diff once a card authorises the
// session, and cleared again by a diff once the session ends.
func TestOpenEVSEIdentify(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus) // no rfid_auth key
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(ctx, t, s.srv.URL, "", "")

	ids, err := c.Identify()
	require.NoError(t, err)
	assert.Nil(t, ids)

	s.send <- `{"rfid_auth":"04A1B2C3"}`
	require.Eventually(t, func() bool {
		ids, err := c.Identify()
		return err == nil && len(ids) == 1
	}, 5*time.Second, 10*time.Millisecond)

	ids, err = c.Identify()
	require.NoError(t, err)
	assert.Equal(t, []string{"04A1B2C3"}, ids)

	s.send <- `{"rfid_auth":""}`
	require.Eventually(t, func() bool {
		ids, err := c.Identify()
		return err == nil && ids == nil
	}, 5*time.Second, 10*time.Millisecond)
}
