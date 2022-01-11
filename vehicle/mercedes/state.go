package mercedes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

var ErrExpiredState = fmt.Errorf("state expired")

const stateValidity = 2 * time.Minute

type State struct {
	key  []byte
	Time time.Time
}

// TODO: Move to another more general place in the repo
func NewState(key []byte) State {
	return State{
		key:  key,
		Time: time.Now(),
	}
}

func (c *State) Encrypt() string {
	plain, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	block, err := aes.NewCipher(c.key)
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
	return base64.URLEncoding.EncodeToString(ciphertext)
}

func Decrypt(enc string, key []byte) (State, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(enc)
	if err != nil {
		return State{}, fmt.Errorf("failed to base64 decode encrypted state: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	var state State
	if err := json.Unmarshal(ciphertext, &state); err != nil {
		return State{}, fmt.Errorf("failed to unmarshal encrypted state: %w", err)
	}

	return state, nil
}

func Validate(rawState string, encryptionKey []byte) error {
	state, err := Decrypt(rawState, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to validate state: %w", err)
	}

	if state.Time.Add(stateValidity).After(time.Now()) {
		return nil
	}

	return ErrExpiredState
}
