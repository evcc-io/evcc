// File: easee_test.go

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

func TestProductUpdate_IgnoreOutdatedProductUpdate(t *testing.T) {
	// Setup mocks and test cases
	e := Easee{
		//	API:  mockAPI,
		obsTime:   make(map[easee.ObservationID]time.Time),
		log:       util.NewLogger("easee"),
		startDone: func() {},
	}

	// Test default init
	id := easee.CHARGER_OP_MODE
	assert.Equal(t, time.Time{}, e.obsTime[id])

	// Test case 1: Normal update
	now := time.Now().Truncate(0) //truncate removes sub nanos
	payload := easee.Observation{
		ID:        easee.CHARGER_OP_MODE,
		Timestamp: now,
		DataType:  easee.Integer,
		Value:     "2",
	}
	jsonPayload, _ := json.Marshal(payload)

	e.ProductUpdate(json.RawMessage(jsonPayload))
	assert.Equal(t, now, e.obsTime[id])
	assert.Equal(t, 2, e.opMode)

	// Test case 2: Outdated update
	outdatedPayload := easee.Observation{
		ID:        easee.CHARGER_OP_MODE,
		Timestamp: now.Add(-5 * time.Second),
		DataType:  easee.Integer,
		Value:     "1",
	}
	outdatedJsonPayload, _ := json.Marshal(outdatedPayload)
	e.ProductUpdate(json.RawMessage(outdatedJsonPayload))

	assert.Equal(t, now, e.obsTime[id])
	assert.Equal(t, 2, e.opMode)
}
