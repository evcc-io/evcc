package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOHPCFLPCCapability(t *testing.T) {
	c, lpc, entity := newOHPCFEGCharger(t)
	c.Caps = implement.New()
	c.connector = eebus.NewConnector()
	c.egLpcEntity = nil
	assert.False(t, api.HasCap[api.Dimmer](c))

	c.egLpcEntity = entity
	for _, supported := range []bool{false, true, false, true} {
		lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(supported).Once()
		assert.Equal(t, supported, api.HasCap[api.Dimmer](c))
	}

	lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true).Once()
	dimmer, ok := api.Cap[api.Dimmer](c)
	require.True(t, ok)
	c.Connect(false)
	assert.False(t, api.HasCap[api.Dimmer](c))
	assert.ErrorIs(t, dimmer.Dim(true), api.ErrNotAvailable)
	_, err := dimmer.Dimmed()
	assert.ErrorIs(t, err, api.ErrNotAvailable)

	c.egLpcEntity = entity
	lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true).Once()
	assert.True(t, api.HasCap[api.Dimmer](c))
}

func TestOHPCFCapabilityFallback(t *testing.T) {
	c, _, _ := newOHPCFEGCharger(t)
	c.Caps = implement.New()
	assert.False(t, api.HasCap[api.PowerLimiter](c))
	limiter := implement.PowerLimiter(func() (float64, float64, error) { return 100, 200, nil })
	implement.Has(c, limiter)
	got, ok := api.Cap[api.PowerLimiter](c)
	require.True(t, ok)
	assert.Same(t, limiter, got)
	assert.False(t, api.HasCap[api.Curtailer](c))
}
