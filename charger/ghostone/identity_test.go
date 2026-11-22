package ghostone

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport_ContextCancellation(t *testing.T) {
	// server that blocks -- simulates slow/unreachable wallbox
	unblock := make(chan struct{})
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
	}))
	srv.StartTLS()
	defer close(unblock)

	log := util.NewLogger("test")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Transport(ctx, log, srv.URL, "user", "pass", transport.Insecure())
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, elapsed, 2*time.Second, "Transport should return promptly when context is cancelled")
}

func TestTransport_Success(t *testing.T) {
	// server that returns a valid JWT-style token
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjo5OTk5OTk5OTk5fQ.signature"))
		w.WriteHeader(http.StatusOK)
	}))
	srv.StartTLS()

	log := util.NewLogger("test")

	tr, err := Transport(context.Background(), log, srv.URL, "user", "pass", transport.Insecure())

	require.NoError(t, err)
	assert.NotNil(t, tr)
}

// TestTransport_RetriesUnauthorized verifies that a token rejected by the
// wallbox triggers a new login and a retry of the original request (#33753)
func TestTransport_RetriesUnauthorized(t *testing.T) {
	const jwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjo5OTk5OTk5OTk5fQ.signature"

	var logins, requests int
	var body []byte

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/jwt/login" {
			logins++
			w.Header().Set("Authorization", "Bearer "+jwt)
			return
		}

		// reject the first request as the wallbox does after a restart
		if requests++; requests == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		body, _ = io.ReadAll(r.Body)
	}))
	srv.StartTLS()

	tr, err := Transport(context.Background(), util.NewLogger("test"), srv.URL, "user", "pass", transport.Insecure())
	require.NoError(t, err)
	require.Equal(t, 1, logins)

	req, err := http.NewRequest(http.MethodPut, srv.URL+"/state", strings.NewReader(`{"value":"threePhase"}`))
	require.NoError(t, err)

	resp, err := (&http.Client{Transport: tr}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, logins, "401 must trigger a new login")
	assert.Equal(t, 2, requests)
	assert.JSONEq(t, `{"value":"threePhase"}`, string(body), "request body must be replayed")
}
