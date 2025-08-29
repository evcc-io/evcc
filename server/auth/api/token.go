package api

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strings"
	"time"

	"github.com/akamensky/base58"
	"golang.org/x/crypto/blake2b"
)

const Prefix = "evcc_"

// New generates a secure token with expiry
func New(secret []byte, ttl time.Duration) (string, error) {
	// Generate random payload
	nonce := make([]byte, 16, 16+8)
	_, _ = rand.Read(nonce)

	expiresAt := time.Now().Add(ttl).Unix()

	// Create payload: nonce + expiry
	expiryBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(expiryBytes, uint64(expiresAt))

	payload := append(nonce, expiryBytes...)

	// Hash with BLAKE2b
	hasher, _ := blake2b.New256(secret)
	hasher.Write(payload)
	hash := hasher.Sum(nil)

	// Combine nonce + expiry + hash
	token := append(payload, hash...)

	return Prefix + base58.Encode(token), nil
}

// Validate verifies token validity
func Validate(token string, secret []byte) error {
	if !strings.HasPrefix(token, Prefix) {
		return errors.New("invalid token format")
	}

	decoded, err := base58.Decode(token[len(Prefix):])
	if err != nil {
		return errors.New("invalid token character")
	}
	if len(decoded) != 56 { // 16 + 8 + 32 bytes
		return errors.New("invalid token length")
	}

	nonce := decoded[:16]
	expiryBytes := decoded[16:24]
	providedHash := decoded[24:]

	// Check expiry
	expiresAt := int64(binary.BigEndian.Uint64(expiryBytes))
	if time.Now().Unix() >= expiresAt {
		return errors.New("token expired")
	}

	// Verify hash
	payload := append(nonce, expiryBytes...)
	hasher, _ := blake2b.New256(secret)
	hasher.Write(payload)
	expectedHash := hasher.Sum(nil)

	// Compare hashes
	if !bytes.Equal(expectedHash, providedHash) {
		return errors.New("invalid token")
	}

	return nil
}
