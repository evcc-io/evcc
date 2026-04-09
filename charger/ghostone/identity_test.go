package ghostone

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenSource_ContextCancellation(t *testing.T) {
	// server that blocks -- simulates slow/unreachable wallbox
	unblock := make(chan struct{})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
	}))
	defer func() {
		close(unblock)
		srv.Close()
	}()

	log := util.NewLogger("test")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := TokenSource(ctx, log, srv.URL, "user", "pass")
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, elapsed, 2*time.Second, "TokenSource should return promptly when context is cancelled")
}

func TestTokenSource_Success(t *testing.T) {
	// server that returns a valid JWT-style token
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjo5OTk5OTk5OTk5fQ.signature"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log := util.NewLogger("test")

	ts, err := TokenSource(context.Background(), log, srv.URL, "user", "pass")

	require.NoError(t, err)
	assert.NotNil(t, ts)
}
