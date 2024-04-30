package psa

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

var mu sync.Mutex

type Identity struct {
	*request.Helper
	oc *oauth2.Config
	oauth2.TokenSource
	mu  sync.Mutex
	log *util.Logger
	sk  string
}

// NewIdentity creates PSA identity
func NewIdentity(log *util.Logger, brand, user string, oc *oauth2.Config, token *oauth2.Token) (*Identity, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	// reuse identity instance
	if instance := getInstance(brand, user); instance != nil {
		return instance, nil
	}

	v := &Identity{
		Helper: request.NewHelper(log),
		log:    log,
		oc:     oc,
		sk:     SettingsKey(brand, user),
	}

	if !token.Valid() {
		v.log.DEBUG.Println("identity.NewIdentity - token not valid - Add expiry")
		token.Expiry = time.Now().Add(time.Duration(10) * time.Second)
	}

	// database token
	if !token.Valid() {
		v.log.DEBUG.Println("identity.NewIdentity - token not valid - database token check started")
		var tok oauth2.Token
		if err := settings.Json(v.sk, &tok); err == nil {
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

	// add instance
	addInstance(brand, user, v)

	return v, nil
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	client := request.NewClient(v.log)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	tok, err := v.oc.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, err
	}

	v.log.DEBUG.Println("identity.RefreshToken - token refreshed")
	v.TokenSource = oauth.RefreshTokenSource(tok, v)
	err = settings.SetJson(v.sk, tok)

	return tok, err
}
