package homeassistant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/evcc-io/evcc/util/request"
)

// SolarForecast is an energy platform solar forecast: energy in Wh by ISO hour
type SolarForecast struct {
	WhHours map[string]float64 `json:"wh_hours"`
}

type wsMessage struct {
	ID          int    `json:"id,omitempty"`
	Type        string `json:"type"`
	AccessToken string `json:"access_token,omitempty"`
	Message     string `json:"message,omitempty"`
	Success     bool   `json:"success,omitempty"`
	Error       *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// websocketURI returns the websocket api endpoint, the supervisor proxy serves it without /api
func (c *Connection) websocketURI() string {
	uri := c.instance.URI()
	path := "/api/websocket"
	if strings.TrimRight(uri, "/") == SupervisorURI {
		path = "/websocket"
	}
	return "ws" + strings.TrimPrefix(uri, "http") + path
}

// SolarForecasts retrieves the Energy dashboard solar forecasts by config entry id
func (c *Connection) SolarForecasts(ctx context.Context) (map[string]SolarForecast, error) {
	token, err := c.instance.Token()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, c.websocketURI(), &websocket.DialOptions{HTTPClient: c.Client})
	if err != nil {
		return nil, err
	}
	defer conn.CloseNow()

	// a week of hourly values per config entry exceeds the 32 KiB default
	conn.SetReadLimit(1 << 20)

	var hello wsMessage
	if err := wsjson.Read(ctx, conn, &hello); err != nil {
		return nil, err
	}
	if hello.Type != "auth_required" {
		return nil, fmt.Errorf("unexpected message: %s", hello.Type)
	}

	if err := wsjson.Write(ctx, conn, wsMessage{Type: "auth", AccessToken: token.AccessToken}); err != nil {
		return nil, err
	}

	var auth wsMessage
	if err := wsjson.Read(ctx, conn, &auth); err != nil {
		return nil, err
	}
	if auth.Type != "auth_ok" {
		return nil, fmt.Errorf("authentication failed: %s", auth.Message)
	}

	if err := wsjson.Write(ctx, conn, wsMessage{ID: 1, Type: "energy/solar_forecast"}); err != nil {
		return nil, err
	}

	var res struct {
		wsMessage
		Result map[string]SolarForecast `json:"result"`
	}
	if err := wsjson.Read(ctx, conn, &res); err != nil {
		return nil, err
	}

	if !res.Success {
		if res.Error != nil {
			return nil, fmt.Errorf("%s: %s", res.Error.Code, res.Error.Message)
		}
		return nil, errors.New("solar forecast failed")
	}

	conn.Close(websocket.StatusNormalClosure, "")

	return res.Result, nil
}
