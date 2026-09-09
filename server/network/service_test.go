package network

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOriginHandlerPublic(t *testing.T) {
	t.Cleanup(func() { settings.SetString(keys.Remote, "") })

	get := func(url string) string {
		rec := httptest.NewRecorder()
		originHandler(func(origin string) string { return origin + "/x" })(rec, httptest.NewRequest(http.MethodGet, url, nil))
		return rec.Body.String()
	}

	assert.JSONEq(t, `[]`, get("/origin?remote"), "disabled remote access yields no remote origin")

	require.NoError(t, settings.SetJson(keys.Remote, map[string]any{"enabled": true, "url": "https://foo.evcc.io/"}))
	assert.JSONEq(t, `["https://foo.evcc.io/x"]`, get("/origin?remote"))
	assert.NotContains(t, get("/origin"), "foo.evcc.io", "local origin unaffected")

	require.NoError(t, settings.SetJson(keys.Remote, map[string]any{"enabled": false, "url": "https://foo.evcc.io/"}))
	assert.JSONEq(t, `[]`, get("/origin?remote"), "disabled remote access yields no remote origin")
	assert.Equal(t, "https://foo.evcc.io", RemoteOrigin(), "origin survives disabling for existing devices")
}
