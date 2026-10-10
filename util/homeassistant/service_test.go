package homeassistant

import (
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
)

func state(id, state, unit, name string) StateResponse {
	var s StateResponse
	s.EntityId = id
	s.State = state
	s.Attributes.UnitOfMeasurement = unit
	s.Attributes.FriendlyName = name
	return s
}

func TestEntityOptions(t *testing.T) {
	states := []StateResponse{
		state("sensor.grid_power", "1234", "W", "Grid Power"),
		state("switch.garden", "on", "", "Garden"),
		state("sensor.backup", "unavailable", "", "Backup"),
	}

	assert.Equal(t, []service.Option{
		{Value: "sensor.backup", Label: "Backup"},
		{Value: "sensor.grid_power", Label: "Grid Power", Hint: "1234 W", Match: true},
	}, entityOptions(states, []string{"sensor"}, []string{"W", "kW"}))

	assert.Len(t, entityOptions(states, nil, nil), 3)
	assert.Empty(t, entityOptions(states, []string{"light"}, nil))
}
