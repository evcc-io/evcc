package providerauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// stand-in auth middleware rejecting every request
func blockAll(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// login/logout are gated, callback stays open but rejects unknown states
func TestSetupGating(t *testing.T) {
	router := mux.NewRouter()
	Setup(router, make(chan util.Param, 1), blockAll)

	get := func(path string) int {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Code
	}

	assert.Equal(t, http.StatusUnauthorized, get("/login?id=x"), "login must be gated")
	assert.Equal(t, http.StatusUnauthorized, get("/logout?id=x"), "logout must be gated")
	assert.Equal(t, http.StatusBadRequest, get("/callback"), "callback must stay open (gated by state token, not session)")
	assert.Equal(t, http.StatusBadRequest, get("/callback?error=x&error_description=y"), "no reflection without state")
}

// fakeProvider hands the login state back as login uri so the test can use it
type fakeProvider struct {
	params url.Values
}

func (p *fakeProvider) Login(state string) (string, *oauth2.DeviceAuthResponse, error) {
	return state, nil, nil
}
func (p *fakeProvider) Logout() error                     { return nil }
func (p *fakeProvider) HandleCallback(q url.Values) error { p.params = q; return nil }
func (p *fakeProvider) Authenticated() bool               { return true }
func (p *fakeProvider) DisplayName() string               { return "fake" }

// the callback redirects to the browser origin the login started from
func TestCallbackRedirectsToOrigin(t *testing.T) {
	h := &Handler{
		log:       util.NewLogger("test"),
		secret:    []byte("0123456789abcdef"),
		providers: make(map[string]api.AuthProvider),
		states:    make(map[string]stateEntry),
		updateC:   make(chan string, 1),
	}
	provider := &fakeProvider{}
	_, err := h.register("fake", provider)
	require.NoError(t, err)

	login := func(returnTo string) string {
		rec := httptest.NewRecorder()
		h.handleLogin(rec, httptest.NewRequest(http.MethodGet, "/login?id=fake&return="+url.QueryEscape(returnTo), nil))
		require.Equal(t, http.StatusOK, rec.Code)

		var res loginResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		return res.LoginUri
	}

	callback := func(query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.handleCallback(rec, httptest.NewRequest(http.MethodGet, "/callback?"+query, nil))
		return rec
	}

	// success lands on the page the login started from
	state := login("http://evcc.local:7070/#/config?vehicle=1")
	rec := callback("state=" + state + "&code=abc")
	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "http://evcc.local:7070/#/config?vehicle=1&callbackCompleted=fake", rec.Header().Get("Location"))
	assert.Equal(t, "abc", provider.params.Get("code"))

	// state is single use
	assert.Equal(t, http.StatusBadRequest, callback("state="+state+"&code=abc").Code)

	// provider errors surface on the same page
	state = login("http://evcc.local:7070/#/config")
	rec = callback("state=" + state + "&error=access_denied&error_description=nope")
	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "http://evcc.local:7070/#/config?callbackError=access_denied%3A+nope", rec.Header().Get("Location"))

	// unusable return urls fall back to the config page
	for _, returnTo := range []string{"", "evcc.local", "javascript:alert(1)", "/#/config?vehicle=1"} {
		state = login(returnTo)
		rec = callback("state=" + state + "&code=abc")
		require.Equal(t, http.StatusFound, rec.Code, returnTo)
		assert.Equal(t, "/#/config?callbackCompleted=fake", rec.Header().Get("Location"), returnTo)
	}
}
