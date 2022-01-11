package vehicle

import "errors"

// ClientCredentials contains OAuth2 client id and secret
type ClientCredentials struct {
	ID, Secret string
}

// Error validates the credentials and returns an error if they are incomplete
func (c *ClientCredentials) Error() error {
	if c.ID == "" {
		return errors.New("missing credentials id")
	}

	if c.Secret == "" {
		return errors.New("missing credentials secret")
	}

	return nil
}

// Tokens contains access and refresh tokens
type Tokens struct {
	Access, Refresh string
}

// Error validates the token and returns an error if they are incomplete
func (t *Tokens) Error() error {
	if t.Access == "" || t.Refresh == "" {
		return errors.New("missing access and/or refresh token, use `evcc token` to create")
	}

	return nil
}
