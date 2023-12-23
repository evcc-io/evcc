package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMqtt(t *testing.T) {
	p, err := NewMqttFromConfig(map[string]any{
		"broker": "localhost:1883",
		"topic":  "price",
	})
	require.NoError(t, err)

	g, err := p.(StringProvider).StringGetter()
	require.NoError(t, err)

	_, err = g()
	require.NoError(t, err)
}
