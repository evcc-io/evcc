package mercedes

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

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
		//AuthURL:   fmt.Sprintf("%s/void/AuthURL", IdUri),
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

	if !token.Valid() {
		v.log.DEBUG.Println("identity.NewIdentity - token not valid - Add expiry")
		token.Expiry = time.Now().Add(time.Duration(10) * time.Second)
	}

	// database token
	if !token.Valid() {
		v.log.DEBUG.Println("identity.NewIdentity - token not valid - database token check started")
		var tok oauth2.Token
		if err := settings.Json(v.settingsKey(), &tok); err == nil {
			token = &tok
		}
	}

	if !token.Valid() && token.RefreshToken != "" {
		v.log.DEBUG.Println("identity.NewIdentity - token not valid - refreshToken started")
		if tok, err := v.RefreshToken(token); err == nil {
			token = tok
		}
	}

	if !token.Valid() {
		return nil, errors.New("token expired")
	}

	v.log.DEBUG.Println("Identity.NewIdentity - Step 6 - OK")
	v.TokenSource = oauth.RefreshTokenSource(token, v)

	// add instance
	addInstance(account, v)

	return v, nil
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("mercedes.%s-%s", v.account, v.region)
}

func (v *Identity) RefreshToken(currenttoken *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	uri := fmt.Sprintf("%s/as/token.oauth2", IdUri)
	data := fmt.Sprintf("grant_type=refresh_token&refresh_token=%s", currenttoken.RefreshToken)

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), mbheaders(true, v.region))

	var res MBToken
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err != nil {
		v.log.DEBUG.Println("RefreshToken - After token - error", err)
		return nil, err
	}
	token := res.GetToken()
	v.TokenSource = oauth.RefreshTokenSource(token, v)
	err = settings.SetJson(v.settingsKey(), token)
	if err != nil {
		v.log.DEBUG.Println("RefreshToken - Can't save token to db - error", err)
		return nil, err
	}
	v.log.DEBUG.Println("Identity.RefreshToken - Token set - ok")
	return token, err
}
