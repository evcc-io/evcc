package vehicle

// ClientCredentials contains OAuth2 client id and secret
type ClientCredentials struct {
	ID, Secret string
}

// Tokens contains access and refresh tokens
type Tokens struct {
	Access, Refresh string
}
