package porsche

import "golang.org/x/oauth2"

type tokenRefresher struct {
	oc     *oauth2.Config
	client *Client
}

func newTokenRefresher(oc *oauth2.Config, login func() error) *tokenRefresher {
	return &tokenRefresher{
		oc: oc,
	}
}

func (t *tokenRefresher) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	return nil, nil
}
