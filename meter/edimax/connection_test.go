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
	statusOn = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="get"><Device.System.Power.State>ON</Device.System.Power.State><NOW_POWER><Device.System.Power.NowPower>118.25</Device.System.Power.NowPower></NOW_POWER></CMD></SMARTPLUG>`
	setupOK  = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="setup">OK</CMD></SMARTPLUG>`
)

func plug(t *testing.T, challenge string, auth func(r *http.Request) bool) (*httptest.Server, *[]string) {
	t.Helper()

	var bodies []string
	state := "ON"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth(r) {
			w.Header().Set("WWW-Authenticate", challenge)
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
	srv, bodies := plug(t, `Basic realm="SP2101W"`, func(r *http.Request) bool {
		u, p, ok := r.BasicAuth()
		return ok && u == "admin" && p == "1234"
	})

	conn := NewConnection(t.Context(), host(srv), "admin", "1234", 0)

	on, err := conn.Enabled()
	require.NoError(t, err)
	assert.True(t, on)

	power, err := conn.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 118.25, power)

	require.NoError(t, conn.Enable(false))
	assert.Contains(t, (*bodies)[len(*bodies)-1], "<Device.System.Power.State>OFF</Device.System.Power.State>")

	on, err = conn.Enabled()
	require.NoError(t, err)
	assert.False(t, on)
}

func TestDigestAuth(t *testing.T) {
	var basic bool
	srv, _ := plug(t, `Digest realm="SP2101W", nonce="abc", qop="auth"`, func(r *http.Request) bool {
		if _, _, ok := r.BasicAuth(); ok {
			basic = true
		}
		return strings.HasPrefix(r.Header.Get("Authorization"), "Digest ")
	})

	power, err := NewConnection(t.Context(), host(srv), "admin", "1234", 0).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 118.25, power)
	assert.False(t, basic, "basic credentials sent to digest device")
}

func TestSetupFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<SMARTPLUG id="edimax"><CMD id="setup">FAILED</CMD></SMARTPLUG>`)
	}))
	defer srv.Close()

	assert.Error(t, NewConnection(t.Context(), host(srv), "", "", 0).Enable(true))
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

	_, err := NewConnection(t.Context(), host(srv), "admin", "1234", 0).CurrentPower()
	assert.Error(t, err)
	assert.False(t, leaked)
}

func TestNoBasicAfterDigest(t *testing.T) {
	var digest, basic bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); ok {
			basic = true
		}
		if !digest {
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Digest ") {
				w.Header().Set("WWW-Authenticate", `Digest realm="SP2101W", nonce="abc", qop="auth"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			digest = true
			_, _ = io.WriteString(w, statusOn)
			return
		}
		// later downgrade attempt
		w.Header().Set("WWW-Authenticate", `Basic realm="SP2101W"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	conn := NewConnection(t.Context(), host(srv), "admin", "1234", 0)

	_, err := conn.CurrentPower()
	require.NoError(t, err)

	_, err = conn.CurrentPower()
	assert.Error(t, err)
	assert.False(t, basic, "basic credentials sent after digest")
}
