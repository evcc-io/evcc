package vw

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"os"

	"golang.org/x/oauth2"
)

const suffix = ".token"

func HashString(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func Restore(key string) *oauth2.Token {
	if r, err := os.Open(key + suffix); err == nil {
		var token oauth2.Token
		if err := gob.NewDecoder(r).Decode(&token); err == nil && token.Valid() {
			return &token
		}
	}
	return nil
}

func Persist(key string, token *oauth2.Token) {
	if w, err := os.OpenFile(key+suffix, os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		_ = gob.NewEncoder(w).Encode(token)
	}
}
