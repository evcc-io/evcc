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
	t.Cleanup(func() {
		settings.SetString(keys.Remote, "")
		config.ExternalUrl = ""
	})

	get := func(url string) string {
		rec := httptest.NewRecorder()
		originHandler(func(origin string) string { return origin + "/x" })(rec, httptest.NewRequest(http.MethodGet, url, nil))
		return rec.Body.String()
	}

	assert.JSONEq(t, `[]`, get("/origin?public"), "disabled remote access yields no remote origin")

	require.NoError(t, settings.SetJson(keys.Remote, map[string]any{"enabled": true, "url": "https://foo.evcc.io/"}))
	assert.JSONEq(t, `["https://foo.evcc.io/x"]`, get("/origin?public"))
	assert.NotContains(t, get("/origin"), "foo.evcc.io", "local origin unaffected")

	require.NoError(t, settings.SetJson(keys.Remote, map[string]any{"enabled": false, "url": "https://foo.evcc.io/"}))
	assert.JSONEq(t, `[]`, get("/origin?public"), "disabled remote access yields no remote origin")
	assert.Empty(t, PublicURL())

	// reverse proxy
	config.ExternalUrl = "http://evcc.local:7070"
	assert.JSONEq(t, `[]`, get("/origin?public"), "http external url is not public")

	config.ExternalUrl = "https://evcc.example.com/"
	assert.JSONEq(t, `["https://evcc.example.com/x"]`, get("/origin?public"))
	assert.Equal(t, "https://evcc.example.com", PublicURL())

	config.ExternalUrl = "https://evcc.example.com/evcc"
	assert.Empty(t, PublicURL(), "public routes must be served at the domain root")

	require.NoError(t, settings.SetJson(keys.Remote, map[string]any{"enabled": true, "url": "https://foo.evcc.io/"}))
	assert.JSONEq(t, `["https://foo.evcc.io/x"]`, get("/origin?public"), "remote access wins")
	assert.Equal(t, "https://foo.evcc.io", PublicURL())
}
