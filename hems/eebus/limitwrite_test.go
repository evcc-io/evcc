package eebus

import (
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newExpiredHeartbeat is a heartbeat that was never set, i.e. missing
func newExpiredHeartbeat() *util.Value[struct{}] {
	return util.NewValue[struct{}](time.Hour)
}

// withLimitMocks wires CS mocks that serve one pending LPC and LPP limit write each
func withLimitMocks(t *testing.T, c *EEBus) (*mocks.CsLPCInterface, *mocks.CsLPPInterface) {
	t.Helper()

	lpc := mocks.NewCsLPCInterface(t)
	lpp := mocks.NewCsLPPInterface(t)
	c.cs = &eebus.ControllableSystem{CsLPCInterface: lpc, CsLPPInterface: lpp}

	lpc.EXPECT().PendingConsumptionLimits().Return(map[model.MsgCounterType]ucapi.LoadLimit{
		1: {Value: 5000, IsActive: true},
	}).Maybe()
	lpp.EXPECT().PendingProductionLimits().Return(map[model.MsgCounterType]ucapi.LoadLimit{
		2: {Value: -1000, IsActive: true},
	}).Maybe()

	return lpc, lpp
}

// TestLimitWrite_InitWithoutHeartbeat covers ATC_COM_NT_CSConnection_001: in "init" a
// limit is only evaluated after a heartbeat, an earlier write is rejected.
func TestLimitWrite_InitWithoutHeartbeat(t *testing.T) {
	c := newTestEEBus(t)
	require.NoError(t, c.run()) // init: failsafe limits apply
	require.Equal(t, StatusFailsafe, c.status)

	lpc, lpp := withLimitMocks(t, c)
	lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), false, "no preceding heartbeat").Once()
	lpp.EXPECT().ApproveOrDenyProductionLimit(model.MsgCounterType(2), false, "no preceding heartbeat").Once()

	c.consumptionWriteApprovalRequired()
	c.productionWriteApprovalRequired()

	assert.False(t, c.controlled)
	assert.Zero(t, c.consumptionLimit)

	// still in init, limited to the failsafe limit
	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status)
	assertConsumptionLimit(t, c, testFailsafeConsumption)
}

// TestLimitWrite_InitAfterHeartbeat covers transition 2 (init -> limited): a heartbeat
// and a following limit take the CS out of "init".
func TestLimitWrite_InitAfterHeartbeat(t *testing.T) {
	c := newTestEEBus(t)
	require.NoError(t, c.run())

	lpc, lpp := withLimitMocks(t, c)
	lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), true, "").Once()
	lpp.EXPECT().ApproveOrDenyProductionLimit(model.MsgCounterType(2), true, "").Once()

	c.updateHeartbeat()
	c.consumptionWriteApprovalRequired()
	c.productionWriteApprovalRequired()
	assert.True(t, c.controlled)

	require.NoError(t, c.run())
	assert.Equal(t, StatusNormal, c.status)
	assertConsumptionLimit(t, c, 5000)
}

// TestLimitWrite_HeartbeatTooOld rejects a write more than 60s after the last heartbeat
// while not controlled (LPC/LPP UC TS section 2.2).
func TestLimitWrite_HeartbeatTooOld(t *testing.T) {
	c := newTestEEBus(t)
	require.NoError(t, c.run())

	c.updateHeartbeat()
	c.heartbeatAt = time.Now().Add(-limitWriteWindow - time.Second)
	c.statusUpdated = c.heartbeatAt.Add(-time.Second)

	lpc, _ := withLimitMocks(t, c)
	lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), false, "no preceding heartbeat").Once()

	c.consumptionWriteApprovalRequired()
	assert.False(t, c.controlled)
}

// TestLimitWrite_FailsafeBeforeHeartbeat covers ATC_COM_PT_CSFS_001: in "failsafe state"
// a limit written before the heartbeat returns is rejected and does not end the failsafe.
func TestLimitWrite_FailsafeBeforeHeartbeat(t *testing.T) {
	c := newTestEEBus(t)

	// controlled operation, then the heartbeat is lost
	c.updateHeartbeat()
	c.controlled = true
	c.status = StatusNormal
	c.heartbeat = newExpiredHeartbeat()
	require.NoError(t, c.run())
	require.Equal(t, StatusFailsafe, c.status)
	require.False(t, c.controlled)

	lpc, _ := withLimitMocks(t, c)
	lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), false, "no preceding heartbeat").Once()
	c.consumptionWriteApprovalRequired()

	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status)
	assertConsumptionLimit(t, c, testFailsafeConsumption)
}

// TestLimitWrite_ControlledWithoutRecentHeartbeat covers ATC_COM_PT_CSLimited_002: while
// controlled, a limit is accepted even if the heartbeat is briefly missing.
func TestLimitWrite_ControlledWithoutRecentHeartbeat(t *testing.T) {
	c := newTestEEBus(t)
	c.controlled = true
	c.heartbeatAt = time.Now().Add(-90 * time.Second)

	lpc, _ := withLimitMocks(t, c)
	lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), true, "").Once()

	c.consumptionWriteApprovalRequired()
	assert.Equal(t, 5000.0, c.consumptionLimit.Value)
}
