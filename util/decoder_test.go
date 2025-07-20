package util

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
