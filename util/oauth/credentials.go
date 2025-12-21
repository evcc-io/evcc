package oauth

import (
	"crypto/sha256"
	"fmt"
)

// CredentialsCacheKey generates a unique cache key from user credentials using SHA256.
// This is used for caching oauth2.TokenSource instances per user to avoid duplicate
// authentication requests when multiple chargers use the same credentials.
func CredentialsCacheKey(user, password string) string {
	h := sha256.New()
	h.Write([]byte(user))
	h.Write([]byte(":"))
	h.Write([]byte(password))
	return fmt.Sprintf("%x", h.Sum(nil))
}
