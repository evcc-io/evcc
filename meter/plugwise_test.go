package meter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPlugwiseFromConfigWrongPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := NewPlugwiseFromConfig(map[string]any{
		"uri":      srv.URL,
		"password": "wrongpass",
	})
	require.EqualError(t, err, "wrong password")
}
