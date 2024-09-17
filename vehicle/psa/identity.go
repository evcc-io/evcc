package psa

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type Identity struct {
	oauth2.TokenSource
	mu      sync.Mutex
	oc      *oauth2.Config
	log     *util.Logger
	subject string
}

// NewIdentity creates PSA identity
func NewIdentity(log *util.Logger, brand, user string, oc *oauth2.Config, token *oauth2.Token) (oauth2.TokenSource, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	// reuse identity instance
	subject := "psa." + strings.ToLower(brand) + "." + strings.ToLower(user)
	if instance := getInstance(subject); instance != nil {
		return instance, nil
	}

	v := &Identity{
		log:     log,
		oc:      oc,
		subject: subject,
	}

	var tok oauth2.Token
	if err := settings.Json(v.subject, &tok); err == nil {
		token = &tok
	}

	if !token.Valid() {
		if tok, err := v.RefreshToken(token); err == nil {
			token = tok
		}
	}

	if !token.Valid() {
		return nil, errors.New("token expired")
	}

	v.TokenSource = oauth.RefreshTokenSource(token, v)

	// add instance
	addInstance(v.subject, v)

	return v, nil
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	client := request.NewClient(v.log)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	tok, err := v.oc.TokenSource(ctx, token).Token()
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			settings.Delete(v.subject)
		}
		return nil, err
	}

	v.TokenSource = oauth.RefreshTokenSource(tok, v)
	err = settings.SetJson(v.subject, tok)

	return tok, err
}
