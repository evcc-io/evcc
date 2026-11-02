package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestDisplayHTTPdReadOnly(t *testing.T) {
	srv := NewDisplayHTTPd("", util.NewParamCache())

	for _, path := range []string{"/api/state", "/api/loadpoints/1/mode/now"} {
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		require.Equal(t, http.StatusMethodNotAllowed, rec.Code, path)
	}

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `id="screen"`)

	stateQuery := `{
		loadpoints: [.loadpoints[] | {
			name, title, mode, alwaysCharge, chargePower, charging, connected, sessionEnergy, chargedEnergy,
			chargeDuration, phasesActive, sessionSolarPercentage, vehicleSoc, vehicleTitle, pvAction,
			pvRemaining, planActive, smartCostActive, effectiveMinCurrent, minCurrent,
			phasesConfigured, chargerPhases1p3p
		}],
		greenShareLoadpoints, pvPower, homePower, batterySoc, batteryPower, gridPower,
		battery: {soc: .battery.soc, power: .battery.power},
		grid: {power: .grid.power}
	}`
	stateURL := url.URL{Path: "/api/state", RawQuery: url.Values{"jq": {stateQuery}}.Encode()}
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, stateURL.String(), nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, strings.HasPrefix(rec.Body.String(), "{"))
	require.Contains(t, rec.Body.String(), `"loadpoints":[]`)

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
