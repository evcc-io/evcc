package charger

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a payload
func createPayload(id easee.ObservationID, timestamp time.Time, dataType easee.DataType, value string) json.RawMessage {
	payload := easee.Observation{
		ID:        id,
		Timestamp: timestamp,
		DataType:  dataType,
		Value:     value,
	}
	out, _ := json.Marshal(payload)
	return json.RawMessage(out)
}

func newEasee() Easee {
	return Easee{
		obsTime:   make(map[easee.ObservationID]time.Time),
		log:       util.NewLogger("easee"),
		startDone: func() {},
	}
}

// Refactored TestProductUpdate_IgnoreOutdatedProductUpdate function to reduce repetition in payload generation
func TestProductUpdate_IgnoreOutdatedProductUpdate(t *testing.T) {
	e := newEasee()

	// Test default init
	assert.Equal(t, time.Time{}, e.obsTime[easee.CHARGER_OP_MODE])

	// Test case 1: Normal update
	now := time.Now().Truncate(0) //truncate removes sub nanos
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, now, easee.Integer, "2"))

	assert.Equal(t, now, e.obsTime[easee.CHARGER_OP_MODE])
	assert.Equal(t, 2, e.opMode)

	// Test case 2: Outdated update
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, now.Add(-5*time.Second), easee.Integer, "1"))

	assert.Equal(t, now, e.obsTime[easee.CHARGER_OP_MODE])
	assert.Equal(t, 2, e.opMode)
}

func TestProductUpdate_IgnoreZeroSessionEnergy(t *testing.T) {
	e := newEasee()

	now := time.Now().Truncate(0)
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, now, easee.Double, "20"))

	assert.Equal(t, now, e.obsTime[easee.SESSION_ENERGY])
	assert.Equal(t, float64(20), e.sessionEnergy)

	t2 := time.Now().Truncate(0)
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, t2, easee.Double, "0.0"))

	//expect observation timestamp updated, value however not
	assert.Equal(t, t2, e.obsTime[easee.SESSION_ENERGY])
	assert.Equal(t, float64(20), e.sessionEnergy)
}

func TestProductUpdate_LifetimeEnergy(t *testing.T) {
	e := newEasee()

	now := time.Now().Truncate(0)
	e.ProductUpdate(createPayload(easee.LIFETIME_ENERGY, now, easee.Double, "20"))

	assert.Equal(t, now, e.obsTime[easee.LIFETIME_ENERGY])
	assert.Equal(t, float64(20), e.totalEnergy)
	assert.Equal(t, float64(20), *e.sessionStartEnergy)

	t2 := time.Now().Truncate(0)
	e.ProductUpdate(createPayload(easee.LIFETIME_ENERGY, t2, easee.Double, "40"))

	assert.Equal(t, t2, e.obsTime[easee.LIFETIME_ENERGY])
	assert.Equal(t, float64(40), e.totalEnergy)
	assert.Equal(t, float64(20), *e.sessionStartEnergy)
}

func TestProductUpdate_SessionStartEnergy(t *testing.T) {
	e := newEasee()

	t_minus_5 := time.Now().Truncate(0).Add(-5 * time.Second)

	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, t_minus_5, easee.Integer, "2"))

	assert.Equal(t, t_minus_5, e.obsTime[easee.CHARGER_OP_MODE])
	assert.Equal(t, 2, e.opMode)

	assert.Nil(t, e.sessionStartEnergy)

	assert.Equal(t, float64(0), e.sessionEnergy)
	assert.NotEqual(t, t_minus_5, e.obsTime[easee.SESSION_ENERGY])
}

func TestProductUpdate_SessionStartEnergyOnBootWithCharge(t *testing.T) {
	e := newEasee()

	t_now := time.Now()

	/*
		see https://github.com/evcc-io/evcc/issues/20384

		replicate these updates on boot
		ProductUpdate EXXXXXXX: (2025-04-05 09:53:55 +0000 UTC) CHARGER_OP_MODE 3
		ProductUpdate EXXXXXXX: (2025-04-05 10:20:00 +0000 UTC) SESSION_ENERGY 14.23585319519043
		ProductUpdate EXXXXXXX: (2025-04-05 10:00:00 +0000 UTC) LIFETIME_ENERGY 4537.070175000014
	*/
	t_opMode, _ := time.Parse(time.DateTime, "2025-04-05 09:53:55")
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, t_opMode, easee.Integer, "3"))

	assert.Equal(t, t_opMode, e.obsTime[easee.CHARGER_OP_MODE])
	assert.Equal(t, 3, e.opMode)

	assert.Nil(t, e.sessionStartEnergy)

	assert.Equal(t, float64(0), e.sessionEnergy)
	assert.NotEqual(t, t_opMode, e.obsTime[easee.SESSION_ENERGY])

	t_sessionEnergy, _ := time.Parse(time.DateTime, "2025-04-05 10:20:00")
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, t_sessionEnergy, easee.Double, "14.23585319519043"))

	// opMode update before resets SessionEnergy, ensure Product Update got ignored
	assert.LessOrEqual(t, t_now, e.obsTime[easee.SESSION_ENERGY])
	assert.Equal(t, float64(0), e.sessionEnergy)

	t_lifetimeEnergy, _ := time.Parse(time.DateTime, "2025-04-05 10:00:00")
	e.ProductUpdate(createPayload(easee.LIFETIME_ENERGY, t_lifetimeEnergy, easee.Double, "4537.070175000014"))

	assert.Equal(t, t_lifetimeEnergy, e.obsTime[easee.LIFETIME_ENERGY])
	assert.Equal(t, float64(4537.070175000014), e.totalEnergy)
	assert.Equal(t, float64(4537.070175000014), *e.sessionStartEnergy)

	energy, _ := e.ChargedEnergy()
	assert.Equal(t, float64(0), energy)
}
