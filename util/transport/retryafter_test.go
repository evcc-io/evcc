package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRateLimited returns a server answering the first limited requests with 429 and the given Retry-After
func testRateLimited(t *testing.T, limited int32, retryAfter string) (*atomic.Int32, string) {
	t.Helper()

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if requests.Add(1) <= limited {
			w.Header().Set("Retry-After", retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	return &requests, srv.URL
}

func TestRetryAfter(t *testing.T) {
	requests, uri := testRateLimited(t, 1, "0")
	client := &http.Client{Transport: RetryAfter(time.Second, nil)}

	start := time.Now()
	resp, err := client.Post(uri, "text/plain", strings.NewReader("body"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), requests.Load())
	assert.GreaterOrEqual(t, time.Since(start), time.Second, "zero cooldown waits the minimum")

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "body", string(b), "body is resent")
}

func TestRetryAfterDate(t *testing.T) {
	requests, uri := testRateLimited(t, 1, time.Now().Add(2*time.Second).UTC().Format(http.TimeFormat))
	client := &http.Client{Transport: RetryAfter(5*time.Second, nil)}

	resp, err := client.Get(uri)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), requests.Load())
}

func TestRetryAfterExceedsMax(t *testing.T) {
	for _, retryAfter := range []string{"5", "", "-1", "9223372037", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)} {
		requests, uri := testRateLimited(t, 1, retryAfter)
		client := &http.Client{Transport: RetryAfter(time.Second, nil)}

		resp, err := client.Get(uri)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "Retry-After %q", retryAfter)
		assert.Equal(t, int32(1), requests.Load(), "Retry-After %q", retryAfter)
	}
}

func TestRetryAfterTimeout(t *testing.T) {
	_, uri := testRateLimited(t, 100, "1")
	client := &http.Client{Transport: RetryAfter(time.Second, nil), Timeout: 100 * time.Millisecond}

	start := time.Now()
	_, err := client.Get(uri)
	require.Error(t, err)
	assert.Less(t, time.Since(start), time.Second, "waiting is bounded by the client timeout")
}
