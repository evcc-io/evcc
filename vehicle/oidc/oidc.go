package oidc

import "time"

// Tokens is an OAuth tokens response
type Tokens struct {
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"` // expiration time in seconds
	IDToken      string    `json:"id_token"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Valid        time.Time // helper to store validity timestamp
}

// OIDCResponse is the well-known OIDC provider response
// https://{oauth-provider-hostname}/.well-known/openid-configuration
type OIDCResponse struct {
	Issuer      string   `json:"issuer"`
	AuthURL     string   `json:"authorization_endpoint"`
	TokenURL    string   `json:"token_endpoint"`
	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}
