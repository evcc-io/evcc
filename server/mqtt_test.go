package server

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMqttNaNInf(t *testing.T) {
	m := &MQTT{}
	assert.Equal(t, "NaN", m.encode(math.NaN()), "NaN not encoded as string")
	assert.Equal(t, "+Inf", m.encode(math.Inf(0)), "Inf not encoded as string")
}
