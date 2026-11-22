package homeassistant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestSolarForecasts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/websocket", r.URL.Path)

		conn, err := websocket.Accept(w, r, nil)
		require.NoError(t, err)
		defer conn.CloseNow()

		ctx := r.Context()
		require.NoError(t, wsjson.Write(ctx, conn, map[string]any{"type": "auth_required"}))

		var auth map[string]any
		require.NoError(t, wsjson.Read(ctx, conn, &auth))
		assert.Equal(t, map[string]any{"type": "auth", "access_token": "secret"}, auth)
		require.NoError(t, wsjson.Write(ctx, conn, map[string]any{"type": "auth_ok"}))

		var req map[string]any
		require.NoError(t, wsjson.Read(ctx, conn, &req))
		assert.Equal(t, map[string]any{"id": 1.0, "type": "energy/solar_forecast"}, req)
		require.NoError(t, wsjson.Write(ctx, conn, map[string]any{
			"id": 1, "type": "result", "success": true,
			"result": map[string]any{
				"entry": map[string]any{"wh_hours": map[string]any{"2026-09-12T12:00:00+00:00": 500}},
			},
		}))

		_, _, _ = conn.Read(ctx)
	}))
	defer srv.Close()

	c, err := NewConnection(util.NewLogger("foo"), srv.URL, "", false)
	require.NoError(t, err)
	c.instance.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "secret"})

	res, err := c.SolarForecasts(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]SolarForecast{
		"entry": {WhHours: map[string]float64{"2026-09-12T12:00:00+00:00": 500}},
	}, res)
}
