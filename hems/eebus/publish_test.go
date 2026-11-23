package eebus

import (
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/mocks"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withPublishMocks(t *testing.T, c *EEBus) (*mocks.CsLPCInterface, *mocks.CsLPPInterface) {
	t.Helper()

	lpc := mocks.NewCsLPCInterface(t)
	lpp := mocks.NewCsLPPInterface(t)
	c.cs = &eebus.ControllableSystem{CsLPCInterface: lpc, CsLPPInterface: lpp}

	return lpc, lpp
}

// TestPublish_DurationExpired covers [LPC-007] and ATC_COM_PT_CSTransition6_001: when the
// duration of an active limit expires the CS deactivates the limit it states to the
// Energy Guard and removes the expired duration.
func TestPublish_DurationExpired(t *testing.T) {
	c := newTestEEBus(t)
	c.heartbeat.Set(struct{}{})
	c.controlled = true

	c.consumptionLimit = ucapi.LoadLimit{Value: 5000, IsActive: true, IsChangeable: true, Duration: time.Millisecond}
	require.NoError(t, c.run())
	assertConsumptionLimit(t, c, 5000)

	time.Sleep(5 * time.Millisecond)
	require.NoError(t, c.run())
	assertConsumptionLimit(t, c, 0)

	lpc, _ := withPublishMocks(t, c)
	lpc.EXPECT().SetConsumptionLimit(ucapi.LoadLimit{Value: 5000, IsChangeable: true}).Return(nil).Once()
	c.publishLimitStates()

	// published once only
	c.publishLimitStates()
}

// TestPublish_Failsafe covers [LPC-009/2] and ATC_COM_PT_CSTransition5/7: in "failsafe
// state" the power limits are stated as deactivated.
func TestPublish_Failsafe(t *testing.T) {
	c := newTestEEBus(t)
	c.consumptionLimit = ucapi.LoadLimit{Value: 5000, IsActive: true, IsChangeable: true}
	c.productionLimit = ucapi.LoadLimit{Value: -1000, IsActive: true, IsChangeable: true}

	// heartbeat missing -> failsafe
	require.NoError(t, c.run())
	require.Equal(t, StatusFailsafe, c.status)
	assert.False(t, c.consumptionLimit.IsActive)
	assert.False(t, c.productionLimit.IsActive)

	lpc, lpp := withPublishMocks(t, c)
	lpc.EXPECT().SetConsumptionLimit(ucapi.LoadLimit{Value: 5000, IsChangeable: true}).Return(nil).Once()
	lpp.EXPECT().SetProductionLimit(ucapi.LoadLimit{Value: -1000, IsChangeable: true}).Return(nil).Once()
	c.publishLimitStates()
}

// TestRun_FailsafeReleaseDoesNotResumeLimit covers transition 10 ([LPC-921]): leaving
// "failsafe state" on a heartbeat without a following limit does not resume the limit
// that was active before the heartbeat was lost.
func TestRun_FailsafeReleaseDoesNotResumeLimit(t *testing.T) {
	c := newTestEEBus(t)
	c.consumptionLimit = ucapi.LoadLimit{Value: 5000, IsActive: true}

	// heartbeat missing -> failsafe
	require.NoError(t, c.run())
	require.Equal(t, StatusFailsafe, c.status)

	// heartbeat back, no limit within 120s
	c.heartbeat.Set(struct{}{})
	require.NoError(t, c.run())
	c.heartbeatReturned = time.Now().Add(-2 * failsafeReleaseTimeout)
	require.NoError(t, c.run())

	assert.Equal(t, StatusNormal, c.status)
	assertConsumptionLimit(t, c, 0)
}
