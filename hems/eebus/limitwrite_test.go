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

// pending limit writes served by withLimitMocks, distinct from the failsafe limits
const (
	testPendingConsumption = 5000.0
	testPendingProduction  = 1500.0
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
		1: {Value: testPendingConsumption, IsActive: true},
	}).Maybe()
	lpp.EXPECT().PendingProductionLimits().Return(map[model.MsgCounterType]ucapi.LoadLimit{
		2: {Value: -testPendingProduction, IsActive: true},
	}).Maybe()

	return lpc, lpp
}

// assertRejectedNotApplied checks that rejected limit writes were not stored and are not
// applied once a heartbeat arrives: the CS keeps its failsafe limits.
func assertRejectedNotApplied(t *testing.T, c *EEBus) {
	t.Helper()

	assert.False(t, c.controlled)
	assert.Zero(t, c.consumptionLimit)
	assert.Zero(t, c.productionLimit)

	c.updateHeartbeat()
	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status)
	assertConsumptionLimit(t, c, testFailsafeConsumption)

	production := c.MaxProductionPower()
	require.NotNil(t, production)
	assert.Equal(t, testFailsafeProduction, *production)
}

// TestLimitWrite_WithoutHeartbeat covers ATC_COM_NT_CSConnection_001: in "init" a limit
// written before a heartbeat is rejected, before and after the first cycle.
func TestLimitWrite_WithoutHeartbeat(t *testing.T) {
	for _, tc := range []struct {
		name       string
		firstCycle bool
	}{
		{"before first cycle", false},
		{"after first cycle", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestEEBus(t)
			if tc.firstCycle {
				require.NoError(t, c.run())
				require.Equal(t, StatusFailsafe, c.status)
			}

			lpc, lpp := withLimitMocks(t, c)
			lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), false, "no preceding heartbeat").Once()
			lpp.EXPECT().ApproveOrDenyProductionLimit(model.MsgCounterType(2), false, "no preceding heartbeat").Once()

			c.consumptionWriteApprovalRequired()
			c.productionWriteApprovalRequired()

			if !tc.firstCycle {
				require.NoError(t, c.run())
			}
			assertRejectedNotApplied(t, c)
		})
	}
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
	assertConsumptionLimit(t, c, testPendingConsumption)
}

// TestLimitWrite_HeartbeatTooOld rejects a write more than 60s after the last heartbeat
// while not controlled (LPC/LPP UC TS section 2.2).
func TestLimitWrite_HeartbeatTooOld(t *testing.T) {
	c := newTestEEBus(t)
	require.NoError(t, c.run())

	c.updateHeartbeat()
	c.heartbeatAt = time.Now().Add(-limitWriteWindow - time.Second)
	c.statusUpdated = c.heartbeatAt.Add(-time.Second)

	lpc, lpp := withLimitMocks(t, c)
	lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), false, "no preceding heartbeat").Once()
	lpp.EXPECT().ApproveOrDenyProductionLimit(model.MsgCounterType(2), false, "no preceding heartbeat").Once()

	c.consumptionWriteApprovalRequired()
	c.productionWriteApprovalRequired()
	assertRejectedNotApplied(t, c)
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

	lpc, lpp := withLimitMocks(t, c)
	lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), false, "no preceding heartbeat").Once()
	lpp.EXPECT().ApproveOrDenyProductionLimit(model.MsgCounterType(2), false, "no preceding heartbeat").Once()

	c.consumptionWriteApprovalRequired()
	c.productionWriteApprovalRequired()

	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status)
	assertRejectedNotApplied(t, c)
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
	assert.Equal(t, testPendingConsumption, c.consumptionLimit.Value)
}

// TestLimitWrite_AfterFailsafeRelease covers "unlimited/autonomous" after LPC-921: a limit
// within 60s after the last heartbeat is evaluated, also if that heartbeat preceded the release.
func TestLimitWrite_AfterFailsafeRelease(t *testing.T) {
	for _, tc := range []struct {
		name         string
		heartbeatAge time.Duration
		accept       bool
		reason       string
	}{
		{"recent heartbeat", 5 * time.Second, true, ""},
		{"heartbeat older than 60s", limitWriteWindow + time.Second, false, "no preceding heartbeat"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestEEBus(t)
			require.NoError(t, c.run())
			require.Equal(t, StatusFailsafe, c.status)

			// heartbeat returns without a following limit, the release window has passed
			c.updateHeartbeat()
			require.NoError(t, c.run())
			c.heartbeatReturned = time.Now().Add(-failsafeReleaseTimeout)
			c.heartbeatAt = time.Now().Add(-tc.heartbeatAge)
			require.NoError(t, c.run())
			require.Equal(t, StatusNormal, c.status)
			require.False(t, c.controlled)

			lpc, lpp := withLimitMocks(t, c)
			lpc.EXPECT().ApproveOrDenyConsumptionLimit(model.MsgCounterType(1), tc.accept, tc.reason).Once()
			lpp.EXPECT().ApproveOrDenyProductionLimit(model.MsgCounterType(2), tc.accept, tc.reason).Once()

			c.consumptionWriteApprovalRequired()
			c.productionWriteApprovalRequired()
			assert.Equal(t, tc.accept, c.controlled)
		})
	}
}
