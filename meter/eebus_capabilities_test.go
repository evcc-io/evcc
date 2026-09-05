package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEEBusLimitCapabilities(t *testing.T) {
	c, lpc, lpp, entity := newEGMeter(t)
	c.egLpcEntity = nil
	c.egLppEntity = nil
	assert.False(t, api.HasCap[api.Dimmer](c))
	assert.False(t, api.HasCap[api.Curtailer](c))
	assert.False(t, api.HasCap[api.Battery](c))

	c.egLpcEntity = entity
	c.egLppEntity = entity
	for _, tc := range []struct{ lpc, lpp bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(tc.lpc).Once()
		lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(tc.lpp).Once()
		assert.Equal(t, tc.lpc, api.HasCap[api.Dimmer](c))
		assert.Equal(t, tc.lpp, api.HasCap[api.Curtailer](c))
	}

	lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true).Once()
	dimmer, ok := api.Cap[api.Dimmer](c)
	require.True(t, ok)
	lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true).Once()
	curtailer, ok := api.Cap[api.Curtailer](c)
	require.True(t, ok)
	c.Connect(false)
	assert.False(t, api.HasCap[api.Dimmer](c))
	assert.False(t, api.HasCap[api.Curtailer](c))
	assert.ErrorIs(t, dimmer.Dim(true), api.ErrNotAvailable)
	assert.ErrorIs(t, curtailer.SetCurtailPercent(0), api.ErrNotAvailable)
	_, err := dimmer.Dimmed()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
	_, err = curtailer.CurtailedPercent()
	assert.ErrorIs(t, err, api.ErrNotAvailable)

	c.egLpcEntity = entity
	c.egLppEntity = entity
	lpc.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPCLimit).Return(true).Once()
	lpp.EXPECT().IsScenarioAvailableAtEntity(entity, eebus.LPPLimit).Return(true).Once()
	assert.True(t, api.HasCap[api.Dimmer](c))
	assert.True(t, api.HasCap[api.Curtailer](c))
}
