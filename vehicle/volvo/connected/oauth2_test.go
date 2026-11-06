package connected

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestOAuthConfigEndpoint(t *testing.T) {
	log := util.NewLogger("test")

	for _, tc := range []struct {
		id       string
		expected string
	}{
		{"dc-abc123", legacyEndpoint.TokenURL},
		{"vid-dc-abc123", endpoint.TokenURL},
		{"abc123", endpoint.TokenURL},
	} {
		assert.Equal(t, tc.expected, OAuthConfig(log, tc.id, "secret", "").Endpoint.TokenURL, tc.id)
	}
}
