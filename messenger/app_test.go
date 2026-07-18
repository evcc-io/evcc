package messenger

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestValidPushToken(t *testing.T) {
	assert.True(t, ValidPushToken("ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]"))
	assert.False(t, ValidPushToken(""))
	assert.False(t, ValidPushToken("foo"))
	assert.False(t, ValidPushToken("ExponentPushToken[unterminated"))
	assert.False(t, ValidPushToken("ExponentPushToken["+string(make([]byte, maxTokenLen))+"]"))
}

func TestAppPushRegister(t *testing.T) {
	m := &AppPush{log: util.NewLogger("test")}

	m.Register("ExponentPushToken[a]")
	m.Register("ExponentPushToken[a]") // duplicate
	m.Register("ExponentPushToken[b]")
	assert.Equal(t, []string{"ExponentPushToken[a]", "ExponentPushToken[b]"}, m.tokens)

	m.Unregister("ExponentPushToken[a]")
	assert.Equal(t, []string{"ExponentPushToken[b]"}, m.tokens)

	m.Unregister("ExponentPushToken[unknown]")
	assert.Equal(t, []string{"ExponentPushToken[b]"}, m.tokens)
}
