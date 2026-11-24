package server

import (
	"io"
	"iter"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/stretchr/testify/assert"
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

type siteStub struct{ site.API }

func (siteStub) ActiveLoadpoints() iter.Seq2[int, loadpoint.API] {
	return func(func(int, loadpoint.API) bool) {}
}

func (siteStub) GetBufferSoc() float64 { return 0 }

func (siteStub) SetBufferSoc(float64) error { return nil }

type authStub struct{ auth.Auth }

func (authStub) GetAuthMode() auth.AuthMode { return auth.Enabled }

func (authStub) IsAdminPasswordValid(string) bool { return false }

func TestCrossOrigin(t *testing.T) {
	srv := NewHTTPd("", NewSocketHub(), Customization{})
	srv.RegisterSiteHandlers(siteStub{})
	srv.Router().Methods(http.MethodPost).Path("/api/auth/login").Handler(loginHandler(authStub{}))

	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	const (
		write   = "/api/buffersoc/50"
		login   = "/api/auth/login"
		foreign = "https://attacker.example"
	)

	tc := []struct {
		name, method, path string
		header             http.Header
		status             int
	}{
		{"write without origin", http.MethodPost, write, nil, http.StatusOK},
		{"write same origin", http.MethodPost, write, http.Header{"Origin": {ts.URL}}, http.StatusOK},
		{"write foreign origin", http.MethodPost, write, http.Header{"Origin": {foreign}}, http.StatusForbidden},
		{"write cross-site", http.MethodPost, write, http.Header{"Sec-Fetch-Site": {"cross-site"}}, http.StatusForbidden},
		{"login same origin", http.MethodPost, login, http.Header{"Origin": {ts.URL}}, http.StatusUnauthorized},
		{"login foreign origin", http.MethodPost, login, http.Header{"Origin": {foreign}}, http.StatusForbidden},
		{"preflight", http.MethodOptions, write, http.Header{"Origin": {foreign}, "Access-Control-Request-Method": {http.MethodPost}}, http.StatusNotFound},
		{"websocket same origin", http.MethodGet, "/ws", http.Header{"Origin": {ts.URL}}, http.StatusSwitchingProtocols},
		{"websocket foreign origin", http.MethodGet, "/ws", http.Header{"Origin": {foreign}}, http.StatusForbidden},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.path == login {
				body = strings.NewReader(`{"password":"guess"}`)
			}

			req, err := http.NewRequest(tc.method, ts.URL+tc.path, body)
			require.NoError(t, err)

			maps.Copy(req.Header, tc.header)
			if tc.path == "/ws" {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Sec-WebSocket-Version", "13")
				req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.status, resp.StatusCode)
			assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		})
	}
}
