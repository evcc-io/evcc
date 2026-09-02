package psa

import (
	"context"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type Identity struct {
	oauth2.TokenSource
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

	ts, err := oauth.PersistentTokenSource(log, v.subject, token, v.refreshToken)
	if err != nil {
		return nil, err
	}

	v.TokenSource = ts

	// add instance
	addInstance(v.subject, v)

	return v, nil
}

func (v *Identity) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	client := request.NewClient(v.log)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	return v.oc.TokenSource(ctx, token).Token()
}
