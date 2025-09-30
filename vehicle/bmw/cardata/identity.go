package cardata

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const ApiURL = "https://api-cardata.bmwgroup.com"

var Config = oauth2.Config{
	Scopes: []string{"authenticate_user", "openid", "cardata:streaming:read", "cardata:api:read"},
	Endpoint: oauth2.Endpoint{
		DeviceAuthURL: "https://customer.bmwgroup.com/gcdm/oauth/device/code",
		TokenURL:      "https://customer.bmwgroup.com/gcdm/oauth/token",
	},
}

type Identity struct {
	*request.Helper
	*oauth2.Config
	oauth2.TokenSource
	log *util.Logger
}

// NewIdentity creates BMW/Mini Cardata identity
func NewIdentity(ctx context.Context, log *util.Logger, config *oauth2.Config) (*Identity, error) {
	v := &Identity{
		Helper: request.NewHelper(log),
		Config: config,
		log:    log,
	}

	var token *oauth2.Token

	var cardataToken Token
	if err := settings.Json(v.settingsKey(), &cardataToken); err == nil {
		v.log.DEBUG.Println("database token found")

		token = cardataToken.TokenEx()

		ctx := context.WithValue(ctx, oauth2.HTTPClient, v.Helper.Client)
		v.TokenSource = &PersistingTokenSource{
			TokenSource: Config.TokenSource(ctx, token),
			Persist:     v.storeToken,
		}
	} else {
		v.log.DEBUG.Println("no database token found, login required")

		// TODO
		return nil, errors.New("missing token")
	}

	return v, nil
}

// func (v *Identity) Login() (oauth2.TokenSource, error) {
// 	// database token
// 	var tok Token
// 	if err := settings.Json(v.settingsKey(), &tok); err == nil {
// 		v.log.DEBUG.Println("identity.Login - database token found")
// 		tok, err := v.RefreshToken(&tok)
// 		if err == nil {
// 			ts := oauth2.ReuseTokenSourceWithExpiry(tok, oauth.RefreshTokenSource(tok, v), 15*time.Minute)
// 			return ts, nil
// 		}
// 		v.log.DEBUG.Println("identity.Login - database token invalid. Proceeding to login via user, password and captcha.")
// 	} else {
// 		v.log.DEBUG.Println("identity.Login - no database token found. Proceeding to login via user, password and captcha.")
// 	}

// }

func (v *Identity) storeToken(token *oauth2.Token) error {
	cardataToken := &Token{
		Token:   token,
		IdToken: tokenExtra(token, "id_token"),
		Gcid:    tokenExtra(token, "gcid"),
	}

	return settings.SetJson(v.settingsKey(), cardataToken)
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("cardata-%s", v.Config.ClientID)
}
