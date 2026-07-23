package messenger

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// token of exactly the given total length
func tokenOfLen(l int) string {
	return tokenPrefix + strings.Repeat("x", l-len(tokenPrefix)-len(tokenSuffix)) + tokenSuffix
}

func TestValidPushToken(t *testing.T) {
	tc := []struct {
		name  string
		token string
		valid bool
	}{
		{"typical", "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]", true},
		{"empty", "", false},
		{"garbage", "foo", false},
		{"unterminated", "ExponentPushToken[unterminated", false},
		{"max length", tokenOfLen(maxTokenLen), true},
		{"too long", tokenOfLen(maxTokenLen + 1), false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.valid, ValidPushToken(tc.token))
		})
	}
}

func TestAppPushRegister(t *testing.T) {
	m := &AppPush{log: util.NewLogger("test")}

	m.Register("ExponentPushToken[a]")
	m.Register("ExponentPushToken[a]") // duplicate
	m.Register("ExponentPushToken[b]")
	m.Register("invalid")
	assert.Equal(t, []string{"ExponentPushToken[a]", "ExponentPushToken[b]"}, m.tokens)

	m.Unregister("ExponentPushToken[a]")
	assert.Equal(t, []string{"ExponentPushToken[b]"}, m.tokens)

	m.Unregister("ExponentPushToken[unknown]")
	assert.Equal(t, []string{"ExponentPushToken[b]"}, m.tokens)
}
