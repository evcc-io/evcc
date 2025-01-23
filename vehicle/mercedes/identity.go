package mercedes

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	mu        sync.Mutex
	log       *util.Logger
	account   string
	region    string
	Sessionid string
}

// OAuth2Config is the OAuth2 configuration for authenticating with the MercedesAPI.
var OAuth2Config = &oauth2.Config{
	//	RedirectURL: fmt.Sprintf("%s/void/RedirectURL", IdUri),
	Endpoint: oauth2.Endpoint{
		// AuthURL:   fmt.Sprintf("%s/void/AuthURL", IdUri),
		TokenURL:  fmt.Sprintf("%s/as/token.oauth2", IdUri),
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: []string{"not_needed", "handled", "elsewhere"},
}

// NewIdentity creates Mercedes identity
func NewIdentity(log *util.Logger, token *oauth2.Token, account string, region string) (*Identity, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	v := &Identity{
		Helper:  request.NewHelper(log),
		log:     log,
		account: account,
		region:  region,
	}

	v.Sessionid = uuid.New().String()
	v.Helper.Transport = &transport.Decorator{
		Base:      v.Helper.Transport, //.NewTripper(log, transport.Insecure()),
		Decorator: transport.DecorateHeaders(mbheaders(true, region)),
	}

	// reuse identity instance
	if instance := getInstance(account); instance != nil {
		v.log.DEBUG.Println("identity.NewIdentity - token found in instance store")
		return instance, nil
	}

	// database token
	v.log.DEBUG.Println("identity.NewIdentity - start database token.")
	var tok oauth2.Token
	if err := settings.Json(v.settingsKey(), &tok); err == nil {
		if !tok.Valid() {
			v.log.DEBUG.Println("identity.NewIdentity - database token found")
			if _, err := v.RefreshToken(&tok); err == nil {
				return v, nil
			} else {
				v.log.DEBUG.Println("identity.NewIdentity - database token refresh failed:", err)
			}
		} else {
			addInstance(account, v)
			return v, nil
		}
	}

	// config token
	v.log.DEBUG.Println("identity.NewIdentity - start config token.")
	if _, err := v.RefreshToken(token); err == nil {
		v.log.DEBUG.Println("identity.NewIdentity - valid config token found.")
		return v, nil
	}

	return nil, errors.New("Token expired. Replace the token in the evcc.yaml with a new one.")
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("mercedes.%s-%s", v.account, v.region)
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	uri := fmt.Sprintf("%s/as/token.oauth2", IdUri)
	req, _ := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), mbheaders(true, v.region))

	var res oauth2.Token
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	tok := util.TokenWithExpiry(&res)
	v.TokenSource = oauth.RefreshTokenSource(tok, v)

	addInstance(v.account, v)
	err := settings.SetJson(v.settingsKey(), tok)

	return tok, err
}
