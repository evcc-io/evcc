package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatPayload(t *testing.T) {
	p, err := NewMqttPluginFromConfig(
		context.TODO(),
		map[string]any{
			"topic":    "some-topic",
			"broker":   "test.mosquitto.org:1884",
			"user":     "rw",
			"password": "readwrite",
			"jq":       ". * 2",
		},
	)
	assert.NoError(t, err)

	{
		payload, err := p.(*Mqtt).FormatPayload(
			"",
			"",
			10,
		)

		assert.NoError(t, err)

		assert.Equal(t, "20", payload)
	}
}
