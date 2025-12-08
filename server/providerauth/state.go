package providerauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const stateValidity = 2 * time.Minute

type State struct {
	Created time.Time `json:"time"`
}

func NewState() State {
	return State{
		Created: time.Now(),
	}
}

// Use base32 to avoid special characters. Changed from base64 with padding for
// compatibility with FordConnect Query in https://github.com/evcc-io/evcc/pull/25462
var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func DecryptState(enc string, key []byte) (*State, error) {
	ciphertext, err := encoding.DecodeString(enc)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode encrypted state: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	var state State
	err = json.Unmarshal(ciphertext, &state)

	return &state, err
}

func (c *State) Encrypt(key []byte) string {
	plain, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plain))

	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plain)

	// convert to base64
	return encoding.EncodeToString(ciphertext)
}

func (c *State) Valid() bool {
	return time.Since(c.Created) <= stateValidity
}
