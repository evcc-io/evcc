package edimax

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	statusOn = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="get"><Device.System.Power.State>ON</Device.System.Power.State><NOW_POWER><Device.System.Power.NowCurrent>0.5432</Device.System.Power.NowCurrent><Device.System.Power.NowPower>118.25</Device.System.Power.NowPower></NOW_POWER></CMD></SMARTPLUG>`
	setupOK  = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="setup">OK</CMD></SMARTPLUG>`
)

func plug(t *testing.T, auth func(r *http.Request) bool) (*httptest.Server, *[]string) {
	t.Helper()

	var bodies []string
	state := "ON"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth(r) {
			w.Header().Set("WWW-Authenticate", `Digest realm="SP2101W", nonce="abc", qop="auth"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		b, _ := io.ReadAll(r.Body)
		body := string(b)
		bodies = append(bodies, body)

		switch {
		case strings.Contains(body, `id="setup"`):
			if strings.Contains(body, ">OFF<") {
				state = "OFF"
			}
			_, _ = io.WriteString(w, setupOK)
		default:
			_, _ = io.WriteString(w, strings.Replace(statusOn, ">ON<", ">"+state+"<", 1))
		}
	}))
	t.Cleanup(srv.Close)

	return srv, &bodies
}

func host(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "http://")
}

func TestStatusAndSwitch(t *testing.T) {
	srv, bodies := plug(t, func(r *http.Request) bool {
		u, p, ok := r.BasicAuth()
		return ok && u == "admin" && p == "1234"
	})

	conn := NewConnection(host(srv), "admin", "1234", 0)

	res, err := conn.Status()
	require.NoError(t, err)
	assert.Equal(t, Status{On: true, Power: 118.25, Current: 0.5432}, res)

	require.NoError(t, conn.Enable(false))
	assert.Contains(t, (*bodies)[1], "<Device.System.Power.State>OFF</Device.System.Power.State>")

	on, err := conn.Enabled()
	require.NoError(t, err)
	assert.False(t, on)
}

func TestDigestAuth(t *testing.T) {
	srv, _ := plug(t, func(r *http.Request) bool {
		return strings.HasPrefix(r.Header.Get("Authorization"), "Digest ")
	})

	power, err := NewConnection(host(srv), "admin", "1234", 0).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 118.25, power)
}

func TestSetupFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<SMARTPLUG id="edimax"><CMD id="setup">FAILED</CMD></SMARTPLUG>`)
	}))
	defer srv.Close()

	assert.Error(t, NewConnection(host(srv), "", "", 0).Enable(true))
}

func TestNoRedirect(t *testing.T) {
	var leaked bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _, leaked = r.BasicAuth()
	}))
	defer other.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	_, err := NewConnection(host(srv), "admin", "1234", 0).Status()
	assert.Error(t, err)
	assert.False(t, leaked)
}
