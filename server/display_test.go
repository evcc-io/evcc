package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestDisplayHTTPdReadOnly(t *testing.T) {
	cache := util.NewParamCache()
	lpIndex := 0
	cache.Add("0.name", util.Param{Loadpoint: &lpIndex, Key: "name", Val: "lp-1"})
	cache.Add("0.title", util.Param{Loadpoint: &lpIndex, Key: "title", Val: "Garage"})
	cache.Add("0.charging", util.Param{Loadpoint: &lpIndex, Key: "charging", Val: true})
	cache.Add("0.password", util.Param{Loadpoint: &lpIndex, Key: "password", Val: "private-loadpoint-data"})
	disabledIndex := 1
	cache.Add("1.name", util.Param{Loadpoint: &disabledIndex, Key: "name", Val: "lp-2"})
	cache.Add("1.title", util.Param{Loadpoint: &disabledIndex, Key: "title", Val: "Disabled"})
	cache.Add("1.disabled", util.Param{Loadpoint: &disabledIndex, Key: "disabled", Val: true})
	cache.Add("pvPower", util.Param{Key: "pvPower", Val: 1500})
	cache.Add("siteTitle", util.Param{Key: "siteTitle", Val: "private-site-data"})
	cache.Add("battery", util.Param{Key: "battery", Val: map[string]any{"soc": 50, "power": 1000, "serial": "private-battery-data"}})
	srv := NewDisplayHTTPd("", cache)

	for _, path := range []string{"/api/state", "/api/loadpoints/1/mode/now"} {
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		require.Equal(t, http.StatusMethodNotAllowed, rec.Code, path)
	}

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `id="screen"`)

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/state?jq=.", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var state map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &state))
	require.Equal(t, float64(1500), state["pvPower"])
	require.NotContains(t, state, "siteTitle")
	require.NotContains(t, rec.Body.String(), "private-site-data")
	require.NotContains(t, rec.Body.String(), "private-loadpoint-data")
	require.NotContains(t, rec.Body.String(), "private-battery-data")

	loadpoints, ok := state["loadpoints"].([]any)
	require.True(t, ok)
	require.Len(t, loadpoints, 2)
	loadpoint, ok := loadpoints[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "lp-1", loadpoint["name"])
	require.Equal(t, "Garage", loadpoint["title"])
	require.Equal(t, true, loadpoint["charging"])
	require.NotContains(t, loadpoint, "password")

	disabledLoadpoint, ok := loadpoints[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, disabledLoadpoint["disabled"])

	battery, ok := state["battery"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(50), battery["soc"])
	require.Equal(t, float64(1000), battery["power"])
	require.NotContains(t, battery, "serial")

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fonts/Montserrat-Bold.woff2", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDisplayHTTPdServesFonts(t *testing.T) {
	dir := t.TempDir()
	fontPath := filepath.Join(dir, "assets")
	require.NoError(t, os.Mkdir(fontPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fontPath, "Montserrat-Bold-hash.woff2"), []byte("font"), 0o600))

	web := assets.Web
	assets.Web = os.DirFS(dir)
	t.Cleanup(func() {
		assets.Web = web
	})

	srv := NewDisplayHTTPd("", util.NewParamCache())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fonts/Montserrat-Bold.woff2", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "font/woff2", rec.Header().Get("Content-Type"))
	require.Equal(t, "font", rec.Body.String())
}
