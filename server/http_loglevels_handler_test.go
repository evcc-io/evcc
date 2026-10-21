package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateLogLevelsHandler(t *testing.T) {
	t.Cleanup(func() { util.LogLevelDefaults("info", nil) })

	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"valid", `{"default":"warn","levels":{"site":"debug"}}`, http.StatusOK},
		{"default only", `{"default":"trace"}`, http.StatusOK},
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

func TestUpdateLogLevelsHandlerKeepsOverrides(t *testing.T) {
	t.Cleanup(func() { util.LogLevelDefaults("info", nil) })
	util.LogLevel("info", map[string]string{"site": "debug"})

	post := func(body string) {
		rec := httptest.NewRecorder()
		updateLogLevelsHandler(func(string, any) {})(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	post(`{"default":"trace"}`)
	level, areas := util.ActiveLogLevel()
	assert.Equal(t, "TRACE", level)
	assert.Equal(t, map[string]string{"site": "DEBUG"}, areas)

	post(`{"default":"trace","levels":{}}`)
	_, areas = util.ActiveLogLevel()
	assert.Empty(t, areas)
}

func TestUpdateLogLevelsHandlerNormalizesCase(t *testing.T) {
	t.Cleanup(func() { util.LogLevelDefaults("info", nil) })

	rec := httptest.NewRecorder()
	body := `{"default":"DEBUG","levels":{"site":"TRACE"}}`
	updateLogLevelsHandler(func(string, any) {})(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	require.Equal(t, http.StatusOK, rec.Code)

	level, areas := util.ActiveLogLevel()
	assert.Equal(t, "DEBUG", level)
	assert.Equal(t, map[string]string{"site": "TRACE"}, areas)
}
