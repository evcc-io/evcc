package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViessmannEquipmentList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/equipment/installations", r.URL.Path)
		require.Equal(t, "true", r.URL.Query().Get("includeGateways"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"id":3242119,"gateways":[{"serial":"7472258009383262"},{"serial":"7472258009383263"}]},
			{"id":1000001,"gateways":[{"serial":"9999999999999999"}]}
		]}`))
	}))
	defer srv.Close()

	res, err := viessmannEquipmentList(srv.Client(), srv.URL)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, 3242119, res[0].ID)
	assert.Len(t, res[0].Gateways, 2)
	assert.Equal(t, "7472258009383262", res[0].Gateways[0].Serial)
}

func TestViessmannEquipmentListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := viessmannEquipmentList(srv.Client(), srv.URL)
	require.Error(t, err)
}

// The service handlers must return an empty JSON list (not an error) when the
// client id is missing or the user has not authorized yet — the config UI
// polls them while the form is being filled.
func TestViessmannServiceHandlersUnauthorized(t *testing.T) {
	for _, path := range []string{"/installations", "/gateways"} {
		req := httptest.NewRequest(http.MethodGet, path+"?clientid=&redirecturi=", nil)
		w := httptest.NewRecorder()
		viessmannServiceMux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, path)
		assert.JSONEq(t, "[]", w.Body.String(), path)
	}
}
