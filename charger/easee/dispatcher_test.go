package easee

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

// suppress unused import errors for imports needed in later tasks
var (
	_ = fmt.Sprintf
	_ = http.StatusOK
	_ = api.ErrTimeout
	_ = httpmock.NewMockTransport
)

func newTestDispatcher(t *testing.T) *CommandDispatcher {
	t.Helper()
	log := util.NewLogger("test")
	h := request.NewHelper(log)
	h.Client.Timeout = 500 * time.Millisecond
	return NewCommandDispatcher(h, log, 500*time.Millisecond)
}

func TestDispatcher_Dispatch_Rogue(t *testing.T) {
	d := newTestDispatcher(t)
	assert.NotPanics(t, func() {
		d.Dispatch(SignalRCommandResponse{
			SerialNumber: "EH123456",
			Ticks:        999999999,
			WasAccepted:  true,
		})
	})
}

func TestDispatcher_Dispatch_ExpectedOrphan(t *testing.T) {
	d := newTestDispatcher(t)
	d.ExpectOrphan(CIRCUIT_MAX_CURRENT_P1)

	assert.NotPanics(t, func() {
		d.Dispatch(SignalRCommandResponse{
			ID:          int(CIRCUIT_MAX_CURRENT_P1),
			Ticks:       111111111,
			WasAccepted: true,
		})
	})

	// Counter consumed — a second call to CancelOrphan returns false
	assert.False(t, d.CancelOrphan(CIRCUIT_MAX_CURRENT_P1))
}

func TestDispatcher_CancelOrphan_Rollback(t *testing.T) {
	d := newTestDispatcher(t)
	d.ExpectOrphan(CIRCUIT_MAX_CURRENT_P1)
	assert.True(t, d.CancelOrphan(CIRCUIT_MAX_CURRENT_P1))
	assert.False(t, d.CancelOrphan(CIRCUIT_MAX_CURRENT_P1))
}

func TestDispatcher_CancelOrphan_DoubleConsume(t *testing.T) {
	d := newTestDispatcher(t)
	d.ExpectOrphan(CIRCUIT_MAX_CURRENT_P1)
	// Dispatch consumes the orphan counter
	d.Dispatch(SignalRCommandResponse{ID: int(CIRCUIT_MAX_CURRENT_P1), Ticks: 111})
	// CancelOrphan now finds nothing
	assert.False(t, d.CancelOrphan(CIRCUIT_MAX_CURRENT_P1))
}
