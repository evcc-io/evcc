package charger

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAPI struct {
	mock.Mock
}

func (m *MockAPI) GetJSON(uri string, data interface{}) error {
	args := m.Called(uri, data)
	return args.Error(0)
}

// Refactored TestProductUpdate_IgnoreOutdatedProductUpdate function to reduce repetition in payload generation
func TestProductUpdate_IgnoreOutdatedProductUpdate(t *testing.T) {
	e := Easee{
		obsTime:   make(map[easee.ObservationID]time.Time),
		log:       util.NewLogger("easee"),
		startDone: func() {},
	}

	// Test default init
	id := easee.CHARGER_OP_MODE
	assert.Equal(t, time.Time{}, e.obsTime[id])

	// Helper function to create a payload
	createPayload := func(timestamp time.Time, value string) []byte {
		payload := easee.Observation{
			ID:        id,
			Timestamp: timestamp,
			DataType:  easee.Integer,
			Value:     value,
		}
		out, _ := json.Marshal(payload)
		return out
	}

	// Test case 1: Normal update
	now := time.Now().Truncate(0) //truncate removes sub nanos
	jsonPayload := createPayload(now, "2")
	e.ProductUpdate(json.RawMessage(jsonPayload))
	assert.Equal(t, now, e.obsTime[id])
	assert.Equal(t, 2, e.opMode)

	// Test case 2: Outdated update
	outdatedPayload := createPayload(now.Add(-5*time.Second), "1")
	e.ProductUpdate(json.RawMessage(outdatedPayload))

	assert.Equal(t, now, e.obsTime[id])
	assert.Equal(t, 2, e.opMode)
}
