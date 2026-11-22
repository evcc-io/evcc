package dataportal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapError(t *testing.T) {
	// build a *request.StatusError for the given status code
	statusErr := func(code int) error {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))
		defer srv.Close()

		return request.NewHelper(util.NewLogger("test")).GetJSON(srv.URL, nil)
	}

	// missing data becomes ErrNotAvailable
	assert.ErrorIs(t, mapError(statusErr(http.StatusNotFound)), api.ErrNotAvailable)

	// server errors are passed through unchanged
	for _, code := range []int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusForbidden} {
		err := statusErr(code)
		require.Error(t, err)
		assert.Equal(t, err, mapError(err))
	}

	// nil stays nil
	assert.NoError(t, mapError(nil))
}
