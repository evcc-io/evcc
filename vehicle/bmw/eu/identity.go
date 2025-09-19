package bmw

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
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
	log *util.Logger
}

// NewIdentity creates BMW/Mini identity
func NewIdentity(log *util.Logger, config *oauth2.Config) *Identity {
	v := &Identity{
		Helper: request.NewHelper(log),
		Config: config,
		log:    log,
	}

	return v
}

func (v *Identity) Login() (oauth2.TokenSource, error) {
	// database token
	var tok GcidIdToken
	if err := settings.Json(v.settingsKey(), &tok); err == nil {
		v.log.DEBUG.Println("identity.Login - database token found")
		tok, err := v.RefreshToken(&tok)
		if err == nil {
			ts := oauth2.ReuseTokenSourceWithExpiry(tok, oauth.RefreshTokenSource(tok, v), 15*time.Minute)
			return ts, nil
		}
		v.log.DEBUG.Println("identity.Login - database token invalid. Proceeding to login via user, password and captcha.")
	} else {
		v.log.DEBUG.Println("identity.Login - no database token found. Proceeding to login via user, password and captcha.")
	}

}

func (v *Identity) retrieveToken(data url.Values) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/oauth/token", v.region.AuthURI)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  request.FormContent,
		"Authorization": v.region.Token.Authorization,
	})

	var tok oauth2.Token
	if err == nil {
		err = v.DoJSON(req, &tok)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	tokex := util.TokenWithExpiry(&tok)

	err = settings.SetJson(v.settingsKey(), tokex)

	return tokex, err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"refresh_token": {token.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	return v.retrieveToken(data)
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("bmw-eu.%s", v.Config.ClientID)
}
