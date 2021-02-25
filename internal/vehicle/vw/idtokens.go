package vw

import (
	"github.com/andig/evcc/internal/vehicle/oidc"
	"golang.org/x/oauth2"
)

// idTokens is the non-OIDC compliant VW ID token structure
type idTokens struct {
	AccessToken, RefreshToken, IDToken string
}

// AsOIDC converts id tokens to OIDC tokens
func (tokens idTokens) AsOIDC() oidc.Token {
	return oidc.Token{
		Token: oauth2.Token{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
		ExpiresIn: 3600,
	}
}
