package psa

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

var mu sync.Mutex

type Identity struct {
	*request.Helper
	oc *oauth2.Config
	oauth2.TokenSource
	mu    sync.Mutex
	log   *util.Logger
	brand string
	realm string
	vin   string
}

// NewIdentity creates PSA identity
func NewIdentity(log *util.Logger, brand, realm, vin, id, secret string, token *oauth2.Token) (*Identity, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	v := &Identity{
		Helper: request.NewHelper(log),
		log:    log,
		oc: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://api.mpsa.com/api/connectedcar/v2/oauth/authorize",
				TokenURL:  fmt.Sprintf("https://idpcvs.%s/am/oauth2/access_token", brand),
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			Scopes: []string{"openid profile"},
		},
		brand: brand,
		realm: realm,
		vin:   vin,
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

	v.TokenSource = oauth.RefreshTokenSource(token, v)
	return v, nil
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("psa.%s.%s", v.brand, v.vin)
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	headers := map[string]string{
		"Authorization": transport.BasicAuthHeader(v.oc.ClientID, v.oc.ClientSecret),
		"Content-type":  request.FormContent,
	}
	data := url.Values{
		"grant_type":    []string{"refresh_token"},
		"refresh_token": []string{token.RefreshToken},
		"scope":         v.oc.Scopes,
	}

	req, _ := request.New(http.MethodPost, v.oc.Endpoint.TokenURL, strings.NewReader(data.Encode()), headers)

	var res oauth.Token
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	tok := (*oauth2.Token)(&res)
	v.TokenSource = oauth.RefreshTokenSource(tok, v)

	err := settings.SetJson(v.settingsKey(), tok)

	return tok, err
}
