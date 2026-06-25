package meter

import (
	"testing"

	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestEEBus(t *testing.T) {
	acceptable := []string{
		"eebus not configured",
	}

	// Test with explicit grid usage (MGCP)
	values := map[string]any{
		"ski":     "22dd0b546beaa6a720302119c87fc5e0e7ae2e07",
		"ip":      "192.0.2.2",
		"usage":   "grid",
		"timeout": "10s",
	}

	if _, err := NewFromConfig(t.Context(), "eebus", values); err != nil && !test.Acceptable(err, acceptable) {
		t.Error(err)
	}

	// Test without usage parameter (should default to MPC)
	valuesNoUsage := map[string]any{
		"ski":     "22dd0b546beaa6a720302119c87fc5e0e7ae2e07",
		"ip":      "192.0.2.2",
		"timeout": "10s",
	}

	if _, err := NewFromConfig(t.Context(), "eebus", valuesNoUsage); err != nil && !test.Acceptable(err, acceptable) {
		t.Error(err)
	}
}

func TestEEBus_DeviceEntities(t *testing.T) {
	maEntity := spinemocks.NewEntityRemoteInterface(t)
	lpcEntity := spinemocks.NewEntityRemoteInterface(t)
	lppEntity := spinemocks.NewEntityRemoteInterface(t)

	t.Run("all_entities", func(t *testing.T) {
		eb := &EEBus{
			log:         util.NewLogger("test"),
			maEntity:    maEntity,
			egLpcEntity: lpcEntity,
			egLppEntity: lppEntity,
		}
		entities := eb.DeviceEntities()
		assert.Len(t, entities, 3)
	})

	t.Run("partial_entities", func(t *testing.T) {
		eb := &EEBus{
			log:      util.NewLogger("test"),
			maEntity: maEntity,
		}
		entities := eb.DeviceEntities()
		assert.Len(t, entities, 1)
		assert.Equal(t, maEntity, entities[0].Entity)
	})

	t.Run("no_entities", func(t *testing.T) {
		eb := &EEBus{log: util.NewLogger("test")}
		assert.Nil(t, eb.DeviceEntities())
	})
}
