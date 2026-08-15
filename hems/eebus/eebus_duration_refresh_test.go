package eebus

import (
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/stretchr/testify/require"
)

// activeEEBus returns a connected EEBus in StatusNormal with a live heartbeat,
// which is the state an Energy Guard limit write arrives in.
func activeEEBus(t *testing.T) *EEBus {
	t.Helper()

	c := newTestEEBus(t)
	c.Connect(true)
	c.heartbeat.Set(struct{}{})
	c.setStatus(StatusNormal)
	c.heartbeatReturned = time.Now()

	return c
}

// TestRun_ConsumptionLimit_DurationSurvivesRefresh verifies that an Energy Guard
// re-notifying a running limit does not cut its lifetime short.
//
// spine stores a stated TimePeriod as an absolute EndTime, so eebus-go reports
// Duration as the time remaining and every refresh hands evcc a smaller value.
// Measuring that against the original activation would let the two clocks meet
// at half the stated duration.
func TestRun_ConsumptionLimit_DurationSurvivesRefresh(t *testing.T) {
	const (
		limit = 4200.0
		total = 2 * time.Hour
		spent = 61 * time.Minute
	)

	c := activeEEBus(t)
	c.consumptionLimit = ucapi.LoadLimit{Duration: total, IsActive: true, Value: limit}
	c.limitReceived = time.Now()

	require.NoError(t, c.run())
	assertConsumptionLimit(t, c, limit)

	// 61 minutes under limit
	*c.consumptionLimitActivated = time.Now().Add(-spent)

	// the EG re-notifies; updateConsumptionLimit() overwrites Duration with the
	// remaining time reported by eebus-go
	c.setConsumptionLimitData(ucapi.LoadLimit{Duration: total - spent, IsActive: true, Value: limit})

	require.NoError(t, c.run())
	assertConsumptionLimit(t, c, limit) // 59 minutes still to go
}

// TestRun_ProductionLimit_DurationSurvivesRefresh is the LPP mirror.
func TestRun_ProductionLimit_DurationSurvivesRefresh(t *testing.T) {
	const (
		total = 2 * time.Hour
		spent = 61 * time.Minute
	)

	c := activeEEBus(t)
	c.productionLimit = ucapi.LoadLimit{Duration: total, IsActive: true, Value: -500}
	c.limitReceived = time.Now()

	require.NoError(t, c.run())
	assertProductionLimit(t, c, true)

	*c.productionLimitActivated = time.Now().Add(-spent)
	c.setProductionLimitData(ucapi.LoadLimit{Duration: total - spent, IsActive: true, Value: -500})

	require.NoError(t, c.run())
	assertProductionLimit(t, c, true)
}
