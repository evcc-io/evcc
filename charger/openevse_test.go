package charger

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openevse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// full status as sent by the firmware on websocket connect (subset of keys, plus noise keys)
const openevseFullStatus = `{"mode":"STA","amp":16000,"voltage":230,"power":11040,"pilot":16,"state":3,"vehicle":1,"status":"active","elapsed":120,"session_energy":1500,"total_energy":42.5,"temp":false,"manual_override":0}`

// openevseTestServer fakes the firmware's /ws and /claims/{id} endpoints
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

func newTestOpenEVSE(t *testing.T, ctx context.Context, uri, user, password string) *OpenEVSE {
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

func TestOpenEVSEReadsFailUntilConnected(t *testing.T) {
	s := newOpenEVSETestServer(t, "") // websocket accepts but never sends
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")

	time.Sleep(200 * time.Millisecond)
	_, err := c.Status()
	assert.Error(t, err)
	_, err = c.Enabled()
	assert.Error(t, err)
	_, err = c.CurrentPower()
	assert.Error(t, err)
}

func TestOpenEVSEStatusMerge(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")
	waitOpenEVSEConnected(t, c)

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

	// non-object frame (pong) and unknown keys are harmless
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

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")

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

func TestOpenEVSEClaimError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"msg":"Could not make claim"}`))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(t, ctx, srv.URL, "", "")
	assert.Error(t, c.Enable(true))
}

func TestOpenEVSEReconnectReassertsClaim(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")
	waitOpenEVSEConnected(t, c)

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

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")
	waitOpenEVSEConnected(t, c)

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

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")
	waitOpenEVSEConnected(t, c)
	require.NoError(t, c.Enable(true))

	cancel()

	require.Eventually(t, func() bool {
		return s.snapshot().deleted == 1
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, "/claims/262145", s.snapshot().claimPath)
}

func TestOpenEVSEBasicAuth(t *testing.T) {
	s := newOpenEVSETestServer(t, openevseFullStatus)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "admin", "secret")
	waitOpenEVSEConnected(t, c)
	require.NoError(t, c.Enable(true))

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	snap := s.snapshot()
	assert.Equal(t, want, snap.wsAuth, "websocket handshake must carry basic auth")
	assert.Equal(t, want, snap.claimAuth, "claim request must carry basic auth")
}

func TestOpenEVSEInvalidState(t *testing.T) {
	s := newOpenEVSETestServer(t, `{"state":6,"vehicle":1,"status":"active"}`)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")
	waitOpenEVSEConnected(t, c)

	_, err := c.Status()
	assert.ErrorContains(t, err, "invalid status: 6")
}

// TestOpenEVSEGarbageFirstFrame covers Finding 1 (fix round 1): a first frame that
// fails to decode must not mark the connection ready with a zero Status. Only once a
// later frame decodes (fully or with only an UnmarshalTypeError) does the charger
// become readable.
func TestOpenEVSEGarbageFirstFrame(t *testing.T) {
	s := newOpenEVSETestServer(t, "garbage") // not valid JSON
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := newTestOpenEVSE(t, ctx, s.srv.URL, "", "")

	// the garbage frame must not make the charger readable
	time.Sleep(200 * time.Millisecond)
	_, err := c.Status()
	assert.Error(t, err)

	// a subsequent valid frame does
	s.send <- openevseFullStatus

	require.Eventually(t, func() bool {
		_, err := c.Status()
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)

	status, err := c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)
}
