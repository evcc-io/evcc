package api

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenCreateValidateRoundtrip(t *testing.T) {
	secret := make([]byte, 32)
	rand.Read(secret)

	token, err := New(secret, time.Hour)
	require.NoError(t, err)
	require.NoError(t, Validate(token, secret))
}
