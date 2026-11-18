package adax

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Adax remote user client API, see https://adax.no/en/wi-fi/api-development/

// ApiURL is the Adax client api base url
const ApiURL = "https://api-1.adax.no/client-api"

// Room temperatures are in 1/100 °C
type Room struct {
	Id                int64  `json:"id"`
	Name              string `json:"name"`
	HeatingEnabled    *bool  `json:"heatingEnabled"`
	TargetTemperature *int   `json:"targetTemperature"`
	Temperature       *int   `json:"temperature"`
}

// CurrentTemperature returns the current room temperature in °C
func (r Room) CurrentTemperature() (float64, bool) {
	return celsius(r.Temperature)
}

// TargetCelsius returns the target room temperature in °C
func (r Room) TargetCelsius() (float64, bool) {
	return celsius(r.TargetTemperature)
}

func celsius(v *int) (float64, bool) {
	if v == nil {
		return 0, false
	}
	return float64(*v) / 100, true
}

// controlRoom is a room control command, target temperature is ignored while heating is disabled
type controlRoom struct {
	Id                int64 `json:"id"`
	HeatingEnabled    bool  `json:"heatingEnabled"`
	TargetTemperature int   `json:"targetTemperature,omitempty"`
}

// TokenSource returns a token source for the account id and api credential password created in the Adax app.
// Adax requires the credentials on refresh as well, so every refresh uses the password grant.
func TokenSource(ctx context.Context, log *util.Logger, client *http.Client, user, password string) (oauth2.TokenSource, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)

	oc := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL:  ApiURL + "/auth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	ts := oauth.RefreshTokenSource(log, nil, func(*oauth2.Token) (*oauth2.Token, error) {
		return oc.PasswordCredentialsToken(ctx, user, password)
	})

	if _, err := ts.Token(); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	return ts, nil
}

type API struct {
	*request.Helper
}

// NewAPI creates an Adax api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	api := &API{
		Helper: request.NewHelper(log),
	}

	api.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   api.Client.Transport,
	}

	return api
}

// Rooms returns all rooms of the account
func (api *API) Rooms(ctx context.Context) ([]Room, error) {
	req, err := request.New(http.MethodGet, ApiURL+"/rest/v1/content/", nil, request.AcceptJSON)
	if err != nil {
		return nil, err
	}

	var res struct {
		Rooms []Room `json:"rooms"`
	}

	err = api.DoJSON(req.WithContext(ctx), &res)

	return res.Rooms, err
}

// SetHeating switches room heating on with the given target temperature in °C, or off
func (api *API) SetHeating(ctx context.Context, roomId int64, enabled bool, temp float64) error {
	room := controlRoom{Id: roomId, HeatingEnabled: enabled}
	if enabled {
		room.TargetTemperature = int(math.Round(temp * 100))
	}

	req, err := request.New(http.MethodPost, ApiURL+"/rest/v1/control/", request.MarshalJSON(map[string]any{
		"rooms": []controlRoom{room},
	}), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res struct {
		Rooms []struct {
			Id     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"rooms"`
	}

	if err := api.DoJSON(req.WithContext(ctx), &res); err != nil {
		return err
	}

	for _, r := range res.Rooms {
		if r.Id != roomId {
			continue
		}
		if r.Status != "OK" {
			return fmt.Errorf("room %d: %s", r.Id, r.Status)
		}
		return nil
	}

	return fmt.Errorf("room %d: missing control status", roomId)
}
