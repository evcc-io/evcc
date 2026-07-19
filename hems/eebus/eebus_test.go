package eebus

import (
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFailsafeConsumption = 4200.0
	testFailsafeProduction  = 1000.0
	testFailsafeDuration    = 2 * time.Hour
	testProductionNominal   = 2000.0
)

// stubSite implements site.API for testing — only GetGridPower is exercised;
// any other call would dereference the nil embedded interface and panic.
type stubSite struct {
	site.API
	gridPower float64
}

func (s *stubSite) GetGridPower() float64 { return s.gridPower }

// newTestEEBus builds a minimally-wired EEBus suitable for exercising run().
// The CS interfaces are nil — the failsafe-exit path under test does not call
// them — and smartgrid persistence is backed by an in-memory SQLite database.
func newTestEEBus(t *testing.T) *EEBus {
	t.Helper()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	failsafeProduction := testFailsafeProduction
	return &EEBus{
		log:                      util.NewLogger("test"),
		site:                     &stubSite{},
		Connector:                eebus.NewConnector(),
		heartbeat:                util.NewValue[struct{}](time.Hour),
		failsafeConsumptionLimit: testFailsafeConsumption,
		failsafeProductionLimit:  &failsafeProduction,
		failsafeDuration:         testFailsafeDuration,
		productionNominalMax:     testProductionNominal,
	}
}

// assertConsumptionLimit checks the HEMS consumption state through the api.HEMS
// surface. Once run() has executed, MaxConsumptionPower is always known (non-nil).
func assertConsumptionLimit(t *testing.T, c *EEBus, limit float64) {
	t.Helper()
	power := c.MaxConsumptionPower()
	require.NotNil(t, power)
	assert.Equal(t, limit, *power)
}

// assertProductionLimit checks the HEMS production state through the api.HEMS surface.
func assertProductionLimit(t *testing.T, c *EEBus, active bool) {
	t.Helper()
	percent := c.CurtailedPercent()
	require.NotNil(t, percent)
	assert.Equal(t, active, *percent < 100)
}

// TestEEBusNoLimitContract verifies api.HEMS's "nil = limiting undefined" contract:
// nil until the controlbox/EnergyGuard first connects, then 0 = no limit.
func TestEEBusNoLimitContract(t *testing.T) {
	c := newTestEEBus(t)

	require.Nil(t, c.MaxConsumptionPower())
	require.Nil(t, c.MaxProductionPower())

	c.Connect(true)

	assertConsumptionLimit(t, c, 0)
	assertProductionLimit(t, c, false)
}

// TestRun_HeartbeatLost_EntersFailsafe verifies the LPC-911/LPP-911 transition:
// a missing heartbeat in the normal state must apply the configured failsafe
// consumption and production limits.
func TestRun_HeartbeatLost_EntersFailsafe(t *testing.T) {
	c := newTestEEBus(t)
	// heartbeat never Set -> Get() returns ErrTimeout

	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status)
	assertConsumptionLimit(t, c, testFailsafeConsumption)
	assertProductionLimit(t, c, true)
}

// TestRun_FailsafeStaysOnMissingHeartbeat is the LPC-921/LPP-921 fix: when the
// heartbeat is still missing the CS keeps applying the failsafe limit (the
// self-determined protective default for Unlimited-autonomous) and does not
// transition to a no-limit state. The previous implementation transitioned to
// StatusNormal with limit=0 once failsafeDuration elapsed, leaving the system
// unprotected until heartbeat returned.
func TestRun_FailsafeStaysOnMissingHeartbeat(t *testing.T) {
	c := newTestEEBus(t)
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

	c := newTestEEBus(t)
	c.status = StatusFailsafe
	c.statusUpdated = time.Now() // well within failsafeDuration
	c.heartbeat.Set(struct{}{})
	c.consumptionLimit = ucapi.LoadLimit{Value: freshLimit, IsActive: true}

	require.NoError(t, c.run())
	assert.Equal(t, StatusNormal, c.status)
	// Final state is the fresh limit (the LPC-914/1 block re-applies after the release).
	assertConsumptionLimit(t, c, freshLimit)
	assertProductionLimit(t, c, false)
}

// TestRun_HeartbeatReturned_NoFreshLimit covers the LPC-918 release case:
// heartbeat restored but EG has no active limit pending -> exit to normal,
// no limit applied.
func TestRun_HeartbeatReturned_NoFreshLimit(t *testing.T) {
	c := newTestEEBus(t)
	c.status = StatusFailsafe
	c.heartbeat.Set(struct{}{})
	c.consumptionLimit = ucapi.LoadLimit{IsActive: false}

	require.NoError(t, c.run())
	assert.Equal(t, StatusNormal, c.status)
	assertConsumptionLimit(t, c, 0)
	assertProductionLimit(t, c, false)
}

// TestRun_ProductionLimitReleasedEarly verifies that an active production limit
// is released as soon as the EG deactivates it (IsActive=false), without waiting
// for its duration to elapse. The previous code only released on duration expiry,
// so unchecking "Activate" in the control box had no effect until the timer ran
// out (see PR #30284 report).
func TestRun_ProductionLimitReleasedEarly(t *testing.T) {
	c := newTestEEBus(t)
	c.heartbeat.Set(struct{}{})

	// EG activates a production limit with a long duration.
	c.productionLimit = ucapi.LoadLimit{IsActive: true, Duration: time.Hour}
	require.NoError(t, c.run())
	assertProductionLimit(t, c, true)

	// EG deactivates well within the duration -> must release immediately.
	c.productionLimit.IsActive = false
	require.NoError(t, c.run())
	assertProductionLimit(t, c, false)
}

// TestNotConnected verifies that all api.HEMS getters make no statement while
// no upstream controlbox has connected yet.
func TestNotConnected(t *testing.T) {
	c := newTestEEBus(t) // Connector never connected

	assert.Nil(t, c.MaxConsumptionPower())
	assert.Nil(t, c.MaxProductionPower())
	assert.Nil(t, c.CurtailedPercent())
	assert.Nil(t, hems.Dimmed(c))
	assert.Nil(t, hems.Curtailed(c))
}

// TestRun_ProductionLimitWithoutNominalMax verifies an incoming LPP limit
// errors instead of being silently ignored when productionNominalMax is unset (#31469).
func TestRun_ProductionLimitWithoutNominalMax(t *testing.T) {
	c := newTestEEBus(t)
	c.heartbeat.Set(struct{}{})
	c.productionNominalMax = 0

	c.productionLimit = ucapi.LoadLimit{Value: -2200, IsActive: true, Duration: time.Hour}
	require.Error(t, c.run())
	assert.Nil(t, c.CurtailedPercent())
}

// TestRun_ConsumptionLimitReleasedEarly is the LPC mirror of the LPP early-release
// case: an active consumption limit must drop as soon as the EG deactivates it.
func TestRun_ConsumptionLimitReleasedEarly(t *testing.T) {
	c := newTestEEBus(t)
	c.heartbeat.Set(struct{}{})

	// EG activates a consumption limit with a long duration.
	c.consumptionLimit = ucapi.LoadLimit{Value: 3000, IsActive: true, Duration: time.Hour}
	require.NoError(t, c.run())
	assertConsumptionLimit(t, c, 3000)

	// EG deactivates well within the duration -> must release immediately.
	c.consumptionLimit.IsActive = false
	require.NoError(t, c.run())
	assertConsumptionLimit(t, c, 0)
}

// TestEEBusEdgeTriggered verifies that applying a limit (passthrough) only
// happens on a genuine transition, not on every steady-state run().
func TestEEBusEdgeTriggered(t *testing.T) {
	c := newTestEEBus(t)
	c.heartbeat.Set(struct{}{})

	calls := 0
	c.passthrough = func(bool) error { calls++; return nil }
	c.consumptionLimit = ucapi.LoadLimit{Value: 3000, IsActive: true, Duration: time.Hour}

	require.NoError(t, c.run())
	require.NoError(t, c.run())
	require.NoError(t, c.run())

	require.Equal(t, 1, calls, "passthrough must fire once on the edge, not every tick")
	assertConsumptionLimit(t, c, 3000)
}
