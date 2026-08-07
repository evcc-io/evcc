package server

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWriteTimeout ensures a handler exceeding the server's write timeout
// leaves the client without a usable response.
func TestWriteTimeout(t *testing.T) {
	timeout := 200 * time.Millisecond

	srv := NewHTTPd("", nil, "")
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
