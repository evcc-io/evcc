package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestRedacted(t *testing.T) {
	var redacted [][]string

	ts := &redactedTokenSource{
		ts:     oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "access", RefreshToken: "refresh"}),
		redact: func(s ...string) { redacted = append(redacted, s) },
	}

	// the slot is only updated when the token has actually changed
	for range 3 {
		_, err := ts.Token()
		require.NoError(t, err)
	}

	assert.Equal(t, [][]string{{"access", "refresh"}}, redacted)
}
