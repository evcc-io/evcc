package providerauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// stand-in auth middleware rejecting every request
func blockAll(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// login/logout are gated, callback stays open (302 to error page, not middleware 401)
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
	assert.Equal(t, http.StatusFound, get("/callback"), "callback must stay open (gated by state token, not session)")
}
