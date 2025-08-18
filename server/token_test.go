package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenCreateValidateRoundtrip(t *testing.T) {
	token, err := CreateToken(time.Hour)
	require.NoError(t, err)
	require.NoError(t, ValidateToken(token))
}
