package server

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWriteTimeout ensures a handler exceeding the server's write timeout
// leaves the client without a usable response.
func TestWriteTimeout(t *testing.T) {
	timeout := 200 * time.Millisecond

	srv := NewHTTPd("", nil, Customization{})
	srv.WriteTimeout = timeout

	srv.Router().HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * timeout)
		w.Write([]byte(`{"result":"ok"}`))
	}).Methods(http.MethodPost)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })

	resp, err := http.Post("http://"+l.Addr().String()+"/slow", "application/json", nil)
	if err == nil {
		defer resp.Body.Close()
		_, err = io.ReadAll(resp.Body)
	}
	require.Error(t, err, "expected connection to be closed after write timeout")
}

// TestImmutableCache only tags responses that map to a real file, so a 404,
// a directory, or a traversal attempt is not pinned for a year.
func TestImmutableCache(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/index-abc123.js": {Data: []byte("x")},
		"assets/sub/keep.js":     {Data: []byte("y")},
	}
	h := immutableCache(fsys, http.FileServer(http.FS(fsys)))

	const want = "private, max-age=31536000, immutable"

	for _, tc := range []struct {
		path   string
		cached bool
	}{
		{"/assets/index-abc123.js", true},
		{"/assets/missing-def456.js", false}, // 404
		{"/assets/sub", false},               // directory
		{"/assets/sub/", false},              // directory, trailing slash
		{"/assets/../secret", false},         // traversal
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))

			got := rec.Header().Get("Cache-Control")
			if tc.cached {
				require.Equal(t, want, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}
