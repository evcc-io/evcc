package meter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPowerWallFromConfigValidation(t *testing.T) {
	_, err := NewPowerWallFromConfig(map[string]any{"password": "secret"})
	assert.ErrorContains(t, err, "missing usage")

	_, err = NewPowerWallFromConfig(map[string]any{"usage": "battery"})
	assert.ErrorContains(t, err, "missing password")
}
