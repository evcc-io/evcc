package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateLogLevelsHandler(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"valid", `{"default":"warn","levels":{"site":"debug"}}`, http.StatusOK},
		{"no overrides", `{"default":"INFO","levels":{}}`, http.StatusOK},
		{"invalid default", `{"default":"loud","levels":{}}`, http.StatusBadRequest},
		{"invalid area level", `{"default":"info","levels":{"site":"loud"}}`, http.StatusBadRequest},
		{"empty area", `{"default":"info","levels":{" ":"debug"}}`, http.StatusBadRequest},
		{"unknown field", `{"default":"info","level":{}}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			updateLogLevelsHandler(func(string, any) {})(rec, req)
			assert.Equal(t, tc.status, rec.Code)
		})
	}
}
