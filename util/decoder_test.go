package util

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeEnv(t *testing.T) {
	t.Setenv("TEST_TOKEN", "secret")

	var dst struct {
		User, Password, Token string
	}

	require.NoError(t, DecodeOther(map[string]any{
		"user":     "${env:TEST_TOKEN}suffix",
		"password": "${maxcurrent}",
		"token":    "${env:TEST_TOKEN}",
	}, &dst))

	assert.Equal(t, "${env:TEST_TOKEN}suffix", dst.User)
	assert.Equal(t, "${maxcurrent}", dst.Password)
	assert.Equal(t, "secret", dst.Token)

	require.Error(t, DecodeOther(map[string]any{"token": "${env:TEST_MISSING}"}, &dst))
}

func TestDecodeNil(t *testing.T) {
	var dst struct {
		User, Password string
	}

	decoderConfig := &mapstructure.DecoderConfig{
		Result:           &dst,
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	require.NoError(t, err)

	err = decoder.Decode(map[string]any{
		"user": nil,
	})
	require.NoError(t, err)

	assert.Equal(t, struct {
		User, Password string
	}{}, dst)
}
