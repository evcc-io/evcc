package vw

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
)

// ChallengeVerifier returns a challenge/verifier base64-encoded code combination
func ChallengeVerifier() (string, string, error) {
	bytes := make([]byte, 32)
	n, err := rand.Read(bytes)
	if n != 32 || err != nil {
		return "", "", err
	}

	verifier := base64.RawURLEncoding.EncodeToString(bytes)
	sha := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sha[:])

	return challenge, verifier, nil
}
