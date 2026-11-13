package messenger

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNtfySendLogsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	m, err := NewNtfyFromConfig(map[string]any{
		"uri":       srv.URL + "/topic",
		"authtoken": "secret",
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	ntfy := m.(*Ntfy)
	ntfy.log.ERROR.SetOutput(&buf)

	ntfy.Send("title", "msg")

	assert.Contains(t, buf.String(), "unexpected status: 401 (Unauthorized)")
	assert.Contains(t, buf.String(), "unauthorized")
	assert.NotContains(t, buf.String(), "auth=")
}
