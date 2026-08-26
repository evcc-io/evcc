package custom

import (
	"testing"

	"github.com/evcc-io/evcc/hems/hems"
	"github.com/stretchr/testify/assert"
)

// TestConsumptionOnly verifies that a custom HEMS without curtailment plugin
// makes no curtailment statement while dimming to the configured limit.
func TestConsumptionOnly(t *testing.T) {
	c := &Custom{
		maxConsumptionPower: func() (float64, error) { return 0, nil },
		productionPercent:   100,
	}

	assert.Nil(t, c.CurtailedPercent())
	assert.Nil(t, c.MaxProductionPower())
	assert.Nil(t, hems.Curtailed(c))

	assert.Equal(t, new(0.0), c.MaxConsumptionPower())
	assert.Equal(t, new(false), hems.Dimmed(c))

	c.setConsumptionLimit(1e3)
	assert.Equal(t, new(1e3), c.MaxConsumptionPower())
	assert.Equal(t, new(true), hems.Dimmed(c))

	c.setConsumptionLimit(0)
	assert.Equal(t, new(0.0), c.MaxConsumptionPower())
	assert.Equal(t, new(false), hems.Dimmed(c))
}

// TestCurtailmentOnly verifies that a custom HEMS without consumption plugin
// makes no dimming statement while curtailing production.
func TestCurtailmentOnly(t *testing.T) {
	c := &Custom{
		curtailedPercent:     func() (int64, error) { return 100, nil },
		productionNominalMax: 1e4,
		productionPercent:    100,
	}

	assert.Nil(t, c.MaxConsumptionPower())
	assert.Nil(t, hems.Dimmed(c))

	assert.Equal(t, new(100), c.CurtailedPercent())
	assert.Equal(t, new(0.0), c.MaxProductionPower())
	assert.Equal(t, new(false), hems.Curtailed(c))

	c.setProductionLimit(60)
	assert.Equal(t, new(60), c.CurtailedPercent())
	assert.Equal(t, new(6e3), c.MaxProductionPower())
	assert.Equal(t, new(true), hems.Curtailed(c))
}

// TestInvalidPluginValues verifies that out-of-range values are rejected and
// the previous limits retained.
func TestInvalidPluginValues(t *testing.T) {
	c := &Custom{
		maxConsumptionPower: func() (float64, error) { return -1, nil },
		curtailedPercent:    func() (int64, error) { return 101, nil },
		productionPercent:   100,
	}

	assert.Error(t, c.runDim())
	assert.Equal(t, new(0.0), c.MaxConsumptionPower())

	assert.Error(t, c.runCurtail())
	assert.Equal(t, new(100), c.CurtailedPercent())
}

func TestConfigValidation(t *testing.T) {
	_, err := NewCustom(nil, nil, nil, 0, 0)
	assert.Error(t, err)

	_, err = NewCustom(nil, nil, func() (int64, error) { return 100, nil }, 0, 0)
	assert.Error(t, err)
}
