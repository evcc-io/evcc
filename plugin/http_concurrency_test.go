package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

type concurrencyRoundTripper func(*http.Request) (*http.Response, error)

func (f concurrencyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func concurrencyHTTP(t *testing.T, ctx context.Context, uri string, config map[string]any) *HTTP {
	t.Helper()
	other := map[string]any{"uri": uri}
	if config != nil {
		other["concurrency"] = config
	}
	p, err := NewHTTPPluginFromConfig(ctx, other)
	require.NoError(t, err)
	return p.(*HTTP)
}

func TestHTTPConcurrencyConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config map[string]any
		err    string
	}{
		{"disabled", nil, ""},
		{"default", map[string]any{"group": t.Name() + "/default"}, ""},
		{"missing group", map[string]any{"limit": 1}, "group is required"},
		{"blank group", map[string]any{"group": "  "}, "group is required"},
		{"zero limit", map[string]any{"group": t.Name(), "limit": 0}, "limit must be positive"},
		{"negative limit", map[string]any{"group": t.Name(), "limit": -1}, "limit must be positive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := map[string]any{"uri": "https://example.org"}
			if tc.config != nil {
				other["concurrency"] = tc.config
			}
			p, err := NewHTTPPluginFromConfig(context.Background(), other)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			if tc.config == nil {
				require.Nil(t, p.(*HTTP).limiter)
			} else {
				require.EqualValues(t, 1, p.(*HTTP).limiter.limit)
			}
		})
	}

	group := t.Name() + "/shared"
	p1 := concurrencyHTTP(t, context.Background(), "https://example.org/pv", map[string]any{"group": group})
	p2 := concurrencyHTTP(t, context.Background(), "https://example.org/temperature", map[string]any{"group": group, "limit": 1})
	require.Same(t, p1.limiter, p2.limiter)
	_, err := NewHTTPPluginFromConfig(context.Background(), map[string]any{
		"uri":         "https://example.org",
		"concurrency": map[string]any{"group": group, "limit": 2},
	})
	require.ErrorContains(t, err, "conflicts with existing limit 1")
}

type concurrencyBody struct {
	io.Reader
	release <-chan struct{}
	active  *atomic.Int32
	once    sync.Once
}

func (b *concurrencyBody) Read(p []byte) (int, error) {
	<-b.release
	return b.Reader.Read(p)
}

func (b *concurrencyBody) Close() error {
	b.once.Do(func() { b.active.Add(-1) })
	return nil
}

func TestHTTPConcurrencyRequests(t *testing.T) {
	for _, tc := range []struct {
		name        string
		limit       int
		independent bool
		disabled    bool
		wantActive  int32
	}{
		{"serial", 1, false, false, 1},
		{"parallel", 2, false, false, 2},
		{"independent groups", 1, true, false, 3},
		{"disabled", 0, false, true, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				done := make(chan error, 3)
				var active, peak atomic.Int32
				for i := range 3 {
					group := t.Name()
					if tc.independent {
						group += fmt.Sprint(i)
					}
					var config map[string]any
					if !tc.disabled {
						config = map[string]any{"group": group, "limit": tc.limit}
					}
					p := concurrencyHTTP(t, context.Background(), fmt.Sprintf("https://example.org/forecast/%d", i), config)
					p.Client.Transport = concurrencyRoundTripper(func(req *http.Request) (*http.Response, error) {
						n := active.Add(1)
						for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     make(http.Header),
							Body: &concurrencyBody{
								Reader: strings.NewReader("ok"), release: release, active: &active,
							},
						}, nil
					})
					go func() {
						_, err := p.request(p.url, "")
						done <- err
					}()
				}
				synctest.Wait()
				// Headers have arrived, but bodies are still blocked.
				require.Equal(t, tc.wantActive, active.Load())
				close(release)
				for range 3 {
					require.NoError(t, <-done)
				}
				require.Equal(t, tc.wantActive, peak.Load())
				require.Zero(t, active.Load())
			})
		})
	}
}

func TestHTTPConcurrencyWaiting(t *testing.T) {
	for _, cancelParent := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%t", cancelParent), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				p := concurrencyHTTP(t, ctx, "https://example.org", map[string]any{"group": t.Name()})
				p.Client.Timeout = time.Second
				require.NoError(t, p.limiter.Acquire(ctx, 1))
				defer p.limiter.Release(1)
				var calls atomic.Int32
				p.Client.Transport = concurrencyRoundTripper(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, errors.New("unexpected request")
				})
				if cancelParent {
					cancel()
				}
				start := time.Now()
				_, err := p.request(p.url, "")
				if cancelParent {
					require.ErrorIs(t, err, context.Canceled)
				} else {
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.Equal(t, time.Second, time.Since(start))
				}
				require.Zero(t, calls.Load())
			})
		})
	}
}

func TestHTTPConcurrencySharedTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := concurrencyHTTP(t, context.Background(), "https://example.org", map[string]any{"group": t.Name()})
		p.Client.Timeout = 10 * time.Second
		require.NoError(t, p.limiter.Acquire(context.Background(), 1))
		go func() {
			time.Sleep(6 * time.Second)
			p.limiter.Release(1)
		}()
		var remaining time.Duration
		p.Client.Transport = concurrencyRoundTripper(func(req *http.Request) (*http.Response, error) {
			deadline, _ := req.Context().Deadline()
			remaining = time.Until(deadline)
			return nil, errors.New("upstream failed")
		})
		_, err := p.request(p.url, "")
		require.ErrorContains(t, err, "upstream failed")
		require.Equal(t, 4*time.Second, remaining)
		// Failure must release the slot as well.
		require.True(t, p.limiter.TryAcquire(1))
		p.limiter.Release(1)
	})
}
