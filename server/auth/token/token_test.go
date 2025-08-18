package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenCreateValidateRoundtrip(t *testing.T) {
	token, err := New(time.Hour)
	require.NoError(t, err)
	require.NoError(t, Validate(token))
}
