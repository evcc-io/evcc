package providerauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// blockAll is a stand-in auth middleware that rejects every request.
func blockAll(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// TestSetupGating pins the wiring: login/logout are gated, callback stays open
// (302 to the error page on the invalid state, not a middleware 401).
func TestSetupGating(t *testing.T) {
	router := mux.NewRouter()
	paramC := make(chan util.Param, 1)
	Setup(router, paramC, blockAll)

	srv := httptest.NewServer(router)
	defer srv.Close()

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	get := func(path string) int {
		resp, err := client.Get(srv.URL + path)
		assert.NoError(t, err)
		defer resp.Body.Close()
		return resp.StatusCode
	}

	assert.Equal(t, http.StatusUnauthorized, get("/login?id=x"), "login must be gated")
	assert.Equal(t, http.StatusUnauthorized, get("/logout?id=x"), "logout must be gated")
	assert.Equal(t, http.StatusFound, get("/callback"), "callback must stay open (gated by state token, not session)")
}
