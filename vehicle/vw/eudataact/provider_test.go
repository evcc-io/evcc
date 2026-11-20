package eudataact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnergySoc(t *testing.T) {
	soc, ok := energySoc([]point{
		{Name: FieldEnergyCurrent, Value: "245.0"},
		{Name: FieldEnergyMaximal, Value: "670.0"},
	})
	assert.True(t, ok)
	assert.InDelta(t, 36.57, soc, 0.01)

	_, ok = energySoc([]point{{Name: FieldEnergyCurrent, Value: "245.0"}})
	assert.False(t, ok)

	_, ok = energySoc([]point{
		{Name: FieldEnergyCurrent, Value: "700"},
		{Name: FieldEnergyMaximal, Value: "670"},
	})
	assert.False(t, ok)
}
