package vehicle

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestMazdaIdentityToken(t *testing.T) {
	key := "test." + t.Name()
	t.Cleanup(func() { settings.SetString(key, "") })

	log := util.NewLogger("test")
	o := &mazdaIdentity{log: log, redact: log.RotatingSlot(), subject: "mazda.test", tokenKey: key}

	_, err := o.Token()
	_, ok := errors.AsType[*api.ErrLoginRequired](err)
	assert.True(t, ok, "login required before login")

	require.Error(t, o.finishToken(&oauth2.Token{AccessToken: "access"}), "missing refresh token")
	assert.False(t, o.Authenticated())

	require.NoError(t, o.finishToken(&oauth2.Token{AccessToken: "access", RefreshToken: "refresh"}))

	token, err := o.Token()
	require.NoError(t, err)
	assert.Equal(t, "access", token.AccessToken)

	var persisted oauth2.Token
	require.NoError(t, settings.Json(key, &persisted))
	assert.Equal(t, "refresh", persisted.RefreshToken)

	require.NoError(t, o.Logout())
	assert.False(t, o.Authenticated())
	assert.False(t, settings.Exists(key))
}
