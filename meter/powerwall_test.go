package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestNewPowerWallFromConfigValidation(t *testing.T) {
	_, err := NewPowerWallFromConfig(map[string]any{"password": "secret"})
	assert.ErrorContains(t, err, "missing usage")

	_, err = NewPowerWallFromConfig(map[string]any{"usage": "battery"})
	assert.ErrorContains(t, err, "missing password")
}

func TestNewPowerWallFleetFromConfigValidation(t *testing.T) {
	_, err := NewPowerWallFleetFromConfig(map[string]any{
		"usage":    "battery",
		"password": "secret",
	})
	assert.ErrorContains(t, err, "missing client id")

	_, err = NewPowerWallFleetFromConfig(map[string]any{
		"usage":    "battery",
		"password": "secret",
		"credentials": map[string]any{
			"id": "client",
		},
	})
	assert.ErrorIs(t, err, api.ErrMissingToken)
}
