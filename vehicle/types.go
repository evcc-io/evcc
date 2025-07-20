package vehicle

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/oauth2"
)

// ClientCredentials contains OAuth2 client id and secret
type ClientCredentials struct {
	ID, Secret string
}

// Error validates the credentials and returns an error if they are incomplete
func (c *ClientCredentials) Error() error {
	if c.ID == "" {
		return errors.New("missing client id")
	}

	if c.Secret == "" {
		return errors.New("missing client secret")
	}

	return nil
}

// Tokens contains access and refresh tokens
type Tokens struct {
	Access, Refresh string
}

// Token builds token from credentials and returns an error if they are incomplete
func (t *Tokens) Token() (*oauth2.Token, error) {
	if t.Access == "" && t.Refresh == "" {
		return nil, api.ErrMissingToken
	}

	return &oauth2.Token{
		AccessToken:  t.Access,
		RefreshToken: t.Refresh,
		Expiry:       time.Now(),
	}, nil
}
