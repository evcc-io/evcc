package adax

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestTokenSource(t *testing.T) {
	client := new(http.Client)
	httpmock.ActivateNonDefault(client)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(http.MethodPost, ApiURL+"/auth/token", func(req *http.Request) (*http.Response, error) {
		require.NoError(t, req.ParseForm())
		assert.Equal(t, "password", req.PostForm.Get("grant_type"))
		assert.Equal(t, "123", req.PostForm.Get("username"))
		assert.Equal(t, "secret", req.PostForm.Get("password"))

		return httpmock.NewJsonResponse(http.StatusOK, map[string]any{
			"access_token": "jwt", "token_type": "bearer", "expires_in": 3600,
		})
	})

	ts, err := TokenSource(context.Background(), util.NewLogger("test"), client, "123", "secret")
	require.NoError(t, err)

	token, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "jwt", token.AccessToken)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

func TestAPI(t *testing.T) {
	api := NewAPI(util.NewLogger("test"), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "jwt"}))

	// mock below the oauth transport
	api.Client.Transport.(*oauth2.Transport).Base = httpmock.DefaultTransport
	defer httpmock.Reset()

	httpmock.RegisterResponder(http.MethodGet, ApiURL+"/rest/v1/content/", func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "Bearer jwt", req.Header.Get("Authorization"))

		return httpmock.NewStringResponse(http.StatusOK, `{
			"homes":[{"id":1,"name":"Home"}],
			"rooms":[{"id":10,"homeId":1,"name":"Office","heatingEnabled":true,"targetTemperature":2150,"temperature":1975},{"id":11,"homeId":1,"name":"Attic"}],
			"devices":[{"id":100,"homeId":1,"roomId":10,"name":"Heater","type":"Heater"}]
		}`), nil
	})

	var control map[string]any
	httpmock.RegisterResponder(http.MethodPost, ApiURL+"/rest/v1/control/", func(req *http.Request) (*http.Response, error) {
		var body struct {
			Rooms []map[string]any `json:"rooms"`
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Len(t, body.Rooms, 1)
		control = body.Rooms[0]

		status := "OK"
		if control["id"] == 11.0 {
			status = "NoAccess"
		}

		return httpmock.NewJsonResponse(http.StatusOK, map[string]any{
			"rooms": []map[string]any{{"id": control["id"], "status": status}},
		})
	})

	rooms, err := api.Rooms()
	require.NoError(t, err)
	require.Len(t, rooms, 2)

	temp, ok := rooms[0].CurrentTemperature()
	assert.True(t, ok)
	assert.Equal(t, 19.75, temp)

	target, ok := rooms[0].TargetCelsius()
	assert.True(t, ok)
	assert.Equal(t, 21.5, target)

	_, ok = rooms[1].CurrentTemperature()
	assert.False(t, ok)

	require.NoError(t, api.SetHeating(10, true, 21.3))
	assert.Equal(t, map[string]any{"id": 10.0, "heatingEnabled": true, "targetTemperature": 2130.0}, control)

	require.NoError(t, api.SetHeating(10, false, 21.3))
	assert.Equal(t, map[string]any{"id": 10.0, "heatingEnabled": false}, control)

	assert.EqualError(t, api.SetHeating(11, false, 0), "room 11: NoAccess")
}
