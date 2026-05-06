package eebus

import (
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testFailsafeConsumption = 4200.0
	testFailsafeProduction  = 1000.0
	testFailsafeDuration    = 2 * time.Hour
)

// newTestEEBus builds a minimally-wired EEBus suitable for exercising run().
// The CS interfaces are nil — the failsafe-exit path under test does not call
// them — and smartgrid persistence is backed by an in-memory SQLite database.
func newTestEEBus(t *testing.T, root api.Circuit) *EEBus {
	t.Helper()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	failsafeProduction := testFailsafeProduction
	return &EEBus{
		log:                      util.NewLogger("test"),
		root:                     root,
		heartbeat:                util.NewValue[struct{}](time.Hour),
		failsafeConsumptionLimit: testFailsafeConsumption,
		failsafeProductionLimit:  &failsafeProduction,
		failsafeDuration:         testFailsafeDuration,
	}
}

// expectConsumptionLimit programs the mock circuit to receive a consumption
// limit. limit==0 means "release" (Dim(false), SetMaxPower(0)); >0 means "apply".
func expectConsumptionLimit(c *api.MockCircuit, limit float64) {
	c.EXPECT().Dim(limit > 0)
	c.EXPECT().SetMaxPower(limit)
	c.EXPECT().GetChargePower().Return(0.0)
}

// expectProductionLimit programs the mock circuit for a production-limit
// transition. active=true on a non-zero EG limit; false on release.
func expectProductionLimit(c *api.MockCircuit, active bool) {
	c.EXPECT().Curtail(active)
	c.EXPECT().GetChargePower().Return(0.0)
}

// TestRun_HeartbeatLost_EntersFailsafe verifies the LPC-911/LPP-911 transition:
// a missing heartbeat in the normal state must apply the configured failsafe
// consumption and production limits.
func TestRun_HeartbeatLost_EntersFailsafe(t *testing.T) {
	ctrl := gomock.NewController(t)
	circuit := api.NewMockCircuit(ctrl)
	c := newTestEEBus(t, circuit)
	// heartbeat never Set -> Get() returns ErrTimeout

	expectConsumptionLimit(circuit, testFailsafeConsumption)
	expectProductionLimit(circuit, true)

	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status)
}

// TestRun_FailsafeStaysOnMissingHeartbeat is the LPC-921/LPP-921 fix: when the
// heartbeat is still missing the CS keeps applying the failsafe limit (the
// self-determined protective default for Unlimited-autonomous) and does not
// transition to a no-limit state. The previous implementation transitioned to
// StatusNormal with limit=0 once failsafeDuration elapsed, leaving the system
// unprotected until heartbeat returned.
func TestRun_FailsafeStaysOnMissingHeartbeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	circuit := api.NewMockCircuit(ctrl)
	c := newTestEEBus(t, circuit)
	c.status = StatusFailsafe
	// statusUpdated set in the past beyond failsafeDuration to verify we do not
	// exit failsafe based on the duration alone.
	c.statusUpdated = time.Now().Add(-2 * testFailsafeDuration)
	// heartbeat missing.

	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status, "must stay in failsafe when heartbeat is still missing")
}

// TestRun_HeartbeatReturned_AppliesFreshLimit covers LPC-918/919/920: when
// heartbeat is restored and an EG limit is pending, evcc must leave failsafe
// immediately and apply the freshly received limit. The previous code waited
// for failsafeDuration to elapse and then dropped to a zero limit, ignoring
// the fresh value.
func TestRun_HeartbeatReturned_AppliesFreshLimit(t *testing.T) {
	const freshLimit = 3000.0

	ctrl := gomock.NewController(t)
	circuit := api.NewMockCircuit(ctrl)
	c := newTestEEBus(t, circuit)
	c.status = StatusFailsafe
	c.statusUpdated = time.Now() // well within failsafeDuration
	c.heartbeat.Set(struct{}{})
	c.consumptionLimit = ucapi.LoadLimit{Value: freshLimit, IsActive: true}

	// Exit clears the consumption limit, then the LPC-914/1 block re-applies
	// the fresh value. Production was never active, so it stays at zero.
	expectConsumptionLimit(circuit, 0)
	expectProductionLimit(circuit, false)
	expectConsumptionLimit(circuit, freshLimit)

	require.NoError(t, c.run())
	assert.Equal(t, StatusNormal, c.status)
}

// TestRun_HeartbeatReturned_NoFreshLimit covers the LPC-918 release case:
// heartbeat restored but EG has no active limit pending -> exit to normal,
// no limit applied.
func TestRun_HeartbeatReturned_NoFreshLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	circuit := api.NewMockCircuit(ctrl)
	c := newTestEEBus(t, circuit)
	c.status = StatusFailsafe
	c.heartbeat.Set(struct{}{})
	c.consumptionLimit = ucapi.LoadLimit{IsActive: false}

	// Only the failsafe-exit release runs; the LPC-914/1 block sees no
	// active limit.
	expectConsumptionLimit(circuit, 0)
	expectProductionLimit(circuit, false)

	require.NoError(t, c.run())
	assert.Equal(t, StatusNormal, c.status)
}
