package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Hash creates a SHA256 hash of the given parameters printed as string
func Hash(v ...any) string {
	hash := sha256.New()
	for _, v := range v {
		hash.Write([]byte(fmt.Sprint(v)))
	}
	return hex.EncodeToString(hash.Sum(nil))
}
