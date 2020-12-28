package vw

import "github.com/andig/evcc/vehicle/oidc"

// idTokens is the non-OIDC compliant VW ID token structure
type idTokens struct {
	AccessToken, RefreshToken, IDToken string
}

// AsOIDC converts id tokens to OIDC tokens
func (tokens idTokens) AsOIDC() oidc.Tokens {
	return oidc.Tokens{
		IDToken:      tokens.IDToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    3600,
	}
}
