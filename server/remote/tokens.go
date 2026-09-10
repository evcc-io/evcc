package remote

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// tokenLifetime caps bearer token validity
const tokenLifetime = 24 * time.Hour

type tokenEntry struct {
	user    string
	expires time.Time
}

// IssueToken creates a random bearer token for an authenticated client.
// Lifetime is capped by the client's expiry. Tokens are held in memory only,
// so a restart invalidates them and clients must re-authenticate.
func (r *Remote) IssueToken(username string) (string, time.Time, error) {
	expires := time.Now().Add(tokenLifetime)
	for _, c := range loadClients() {
		if c.Username == username && c.ExpiresAt != nil && c.ExpiresAt.Before(expires) {
			expires = *c.ExpiresAt
		}
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(buf)

	r.mu.Lock()
	defer r.mu.Unlock()

	// drop expired tokens
	now := time.Now()
	for k, e := range r.tokens {
		if now.After(e.expires) {
			delete(r.tokens, k)
		}
	}

	r.tokens[token] = tokenEntry{user: username, expires: expires}

	return token, expires, nil
}

// ValidateToken returns the client username for a valid bearer token.
func (r *Remote) ValidateToken(token string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.tokens[token]
	if !ok || time.Now().After(e.expires) {
		return "", false
	}

	return e.user, true
}
