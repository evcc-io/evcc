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

const (
	testURI    = API + "/chargers/TESTTEST/settings"
	testCmdURI = API + "/chargers/TESTTEST/commands/resume_charging"
)

func newTestDispatcher(t *testing.T) *CommandDispatcher {
	t.Helper()
	log := util.NewLogger("test")
	h := request.NewHelper(log)
	h.Client.Timeout = 500 * time.Millisecond
	return NewCommandDispatcher(h, log, 500*time.Millisecond)
}

// waitForPendingTick blocks until d.pendingTicks contains ticks, or the test fails.
func waitForPendingTick(t *testing.T, d *CommandDispatcher, ticks int64) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for pendingTicks registration for ticks %d", ticks)
		default:
		}
		d.mu.Lock()
		_, ok := d.pendingTicks[ticks]
		d.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func TestDispatcher_Dispatch_Rogue(t *testing.T) {
	// Intentionally triggers a WARN — suppress it to keep test output clean.
	util.LogLevel("error", nil)
	t.Cleanup(func() { util.LogLevel("info", nil) })

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

// --- Send tests ---

func TestDispatcher_Send_HTTP200Sync(t *testing.T) {
	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusOK, ""))

	assert.NoError(t, d.Send(testURI, nil))
}

func TestDispatcher_Send_Noop(t *testing.T) {
	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	// Empty array body → Ticks == 0 → noop
	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusAccepted, "[]"))

	assert.NoError(t, d.Send(testURI, nil))
}

func TestDispatcher_Send_HTTP202_InvalidJSON(t *testing.T) {
	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)

	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusAccepted, "{not-json"))

	assert.Error(t, d.Send(testURI, nil))
}

func TestDispatcher_Send_HTTP202_NonNumericTicks(t *testing.T) {
	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)

	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusAccepted, `[{"ticks":"NaN"}]`))

	assert.Error(t, d.Send(testURI, nil))
}

func TestDispatcher_Send_HTTPError(t *testing.T) {
	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusBadRequest, ""))

	err := d.Send(testURI, nil)
	assert.Error(t, err)
}

func TestDispatcher_Send_TicksMatch(t *testing.T) {
	const ticks int64 = 638798974487432600

	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	body := fmt.Sprintf(`[{"device":"TESTTEST","commandId":48,"ticks":%d}]`, ticks)
	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusAccepted, body))

	go func() {
		waitForPendingTick(t, d, ticks)
		d.Dispatch(SignalRCommandResponse{Ticks: ticks, WasAccepted: true})
	}()

	assert.NoError(t, d.Send(testURI, nil))
}

func TestDispatcher_Send_IDFallback(t *testing.T) {
	const ticks int64 = 638798974487432600
	const obsID = DYNAMIC_CHARGER_CURRENT // ObservationID = 48

	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	body := fmt.Sprintf(`[{"device":"TESTTEST","commandId":%d,"ticks":%d}]`, int(obsID), ticks)
	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusAccepted, body))

	go func() {
		waitForPendingTick(t, d, ticks)
		// Wrong Ticks (T+1), correct ID — triggers the ID fallback path
		d.Dispatch(SignalRCommandResponse{ID: int(obsID), Ticks: ticks + 1, WasAccepted: true})
	}()

	assert.NoError(t, d.Send(testURI, nil))
}

func TestDispatcher_Send_Timeout(t *testing.T) {
	const ticks int64 = 789

	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	body := fmt.Sprintf(`[{"device":"TESTTEST","commandId":48,"ticks":%d}]`, ticks)
	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusAccepted, body))

	// No Dispatch call → Send times out
	assert.ErrorIs(t, d.Send(testURI, nil), api.ErrTimeout)
}

func TestDispatcher_Send_Rejected(t *testing.T) {
	const ticks int64 = 456

	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	body := fmt.Sprintf(`[{"device":"TESTTEST","commandId":48,"ticks":%d}]`, ticks)
	httpmock.RegisterResponder(http.MethodPost, testURI,
		httpmock.NewStringResponder(http.StatusAccepted, body))

	go func() {
		waitForPendingTick(t, d, ticks)
		d.Dispatch(SignalRCommandResponse{Ticks: ticks, WasAccepted: false})
	}()

	err := d.Send(testURI, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rejected")
}

func TestDispatcher_Send_CommandURI(t *testing.T) {
	const ticks int64 = 638798974487432600

	d := newTestDispatcher(t)
	httpmock.ActivateNonDefault(d.helper.Client)
	// per-client mock; no global teardown needed

	// /commands/ endpoint → body is a JSON object, not an array
	body := fmt.Sprintf(`{"device":"TESTTEST","commandId":48,"ticks":%d}`, ticks)
	httpmock.RegisterResponder(http.MethodPost, testCmdURI,
		httpmock.NewStringResponder(http.StatusAccepted, body))

	go func() {
		waitForPendingTick(t, d, ticks)
		d.Dispatch(SignalRCommandResponse{Ticks: ticks, WasAccepted: true})
	}()

	assert.NoError(t, d.Send(testCmdURI, nil))
}
