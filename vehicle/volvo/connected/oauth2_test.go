package connected

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOAuthConfigEndpoint(t *testing.T) {
	for _, tc := range []struct {
		id       string
		expected string
	}{
		{"dc-abc123", legacyEndpoint.TokenURL},
		{"vid-dc-abc123", endpoint.TokenURL},
		{"abc123", endpoint.TokenURL},
	} {
		assert.Equal(t, tc.expected, OAuthConfig(tc.id, "secret", "").Endpoint.TokenURL, tc.id)
	}
}
