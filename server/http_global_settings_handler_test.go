package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

func TestSettingsSetYamlHandlerCriticalPlugin(t *testing.T) {
	const pw = "secret"
	const key = "test_circuits_guard"
	body := "- name: main\n  getmaxcurrent:\n    source: script\n    cmd: echo 1"

	a := fakeAuth{mode: auth.Enabled, password: pw}
	h := settingsSetYamlHandler(key, []map[string]any{}, []config.Named{}, a)

	// session without password is rejected and nothing is persisted
	r := httptest.NewRequest(http.MethodPost, "/circuits", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, r)

	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
	_, err := settings.String(key)
	assert.ErrorIs(t, err, settings.ErrNotFound)

	// valid admin password persists the config
	r = httptest.NewRequest(http.MethodPost, "/circuits", strings.NewReader(body))
	r.Header.Set("X-Admin-Password", pw)
	w = httptest.NewRecorder()
	h(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	got, err := settings.String(key)
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(body), got)
}
