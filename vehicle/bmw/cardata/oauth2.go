package cardata

import (
	"context"
	"encoding/json"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("cardata", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			ClientID string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		return NewOAuth(cc.ClientID, "")
	})
}

func OAuthConfig(clientId string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: clientId,
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: "https://customer.bmwgroup.com/gcdm/oauth/device/code",
			TokenURL:      "https://customer.bmwgroup.com/gcdm/oauth/token",
			AuthStyle:     oauth2.AuthStyleInParams,
		},
		Scopes: []string{
			"authenticate_user",
			"openid",
			"cardata:streaming:read",
			"cardata:api:read",
		},
	}
}

func NewOAuth(clientId, title string) (oauth2.TokenSource, error) {
	oc := OAuthConfig(clientId)

	return auth.NewOAuth(context.Background(), "BMW/Mini", title, oc,
		auth.WithOauthDeviceFlowOption(),
		auth.WithTokenRetrieverOption(func(data string, res *oauth2.Token) error {
			var token Token
			if err := json.Unmarshal([]byte(data), &token); err != nil {
				return err
			}
			*res = *token.TokenEx()
			return nil
		}),
		auth.WithTokenStorerOption(func(token *oauth2.Token) any {
			return Token{
				Token:   token,
				IdToken: TokenExtra(token, "id_token"),
				Gcid:    TokenExtra(token, "gcid"),
			}
		}))
}
