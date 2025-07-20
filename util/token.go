package util

import (
	"time"

	"golang.org/x/oauth2"
)

func TokenWithExpiry(token *oauth2.Token) *oauth2.Token {
	if token != nil && token.Expiry.IsZero() && token.ExpiresIn != 0 {
		token.Expiry = time.Now().Add(time.Second * time.Duration(token.ExpiresIn))
	}
	return token
}
