package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatPayloadMultiply(t *testing.T) {
	p, err := NewMqttPluginFromConfig(
		context.TODO(),
		map[string]any{
			"topic":    "some-topic",
			"broker":   "test.mosquitto.org:1884",
			"user":     "rw",
			"password": "readwrite",
			"jq":       ". * 2",
			"payload":  "${var:%d}",
		},
	)
	assert.NoError(t, err)

	{
		payload, err := setFormattedValue(p.(*Mqtt).payload, "var", 10, p.(*Mqtt).pipeline)
		assert.NoError(t, err)
		assert.Equal(t, "20", payload)
	}	
}

func TestFormatPayloadAdd(t *testing.T) {
	p, err := NewMqttPluginFromConfig(
		context.TODO(),
		map[string]any{
			"topic":    "some-topic",
			"broker":   "test.mosquitto.org:1884",
			"user":     "rw",
			"password": "readwrite",
			"jq":       ". + 1",
			"payload":  "${var:%d}",
		},
	)
	assert.NoError(t, err)

	{
		payload, err := setFormattedValue(p.(*Mqtt).payload, "var", 0, p.(*Mqtt).pipeline)
		assert.NoError(t, err)
		assert.Equal(t, "1", payload)
	}	
}
