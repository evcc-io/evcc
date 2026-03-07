package charger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func newEasee() *Easee {
	log := util.NewLogger("easee")
	e := Easee{
		Helper:          request.NewHelper(log),
		obsTime:         make(map[easee.ObservationID]time.Time),
		pendingTicks:    make(map[int64]chan easee.SignalRCommandResponse),
		pendingByID:     make(map[easee.ObservationID]chan easee.SignalRCommandResponse),
		expectedOrphans: make(map[easee.ObservationID]int),
		log:             log,
		startDone:       func() {},
		obsC:            make(chan easee.Observation),
	}
	e.Client.Timeout = 500 * time.Millisecond //aggressive timeout to accelerate testing
	return &e
}

func TestProductUpdate_IgnoreOutdatedProductUpdate(t *testing.T) {
	e := newEasee()

	// Test default init
	assert.Equal(t, time.Time{}, e.obsTime[easee.CHARGER_OP_MODE])

	// Test case 1: Normal update
	now := time.Now().UTC().Truncate(0) //truncate removes sub nanos
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

	now := time.Now().UTC().Truncate(0)
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, now, easee.Double, "20"))

	assert.Equal(t, now, e.obsTime[easee.SESSION_ENERGY])
	assert.Equal(t, float64(20), e.sessionEnergy)

	t2 := time.Now().UTC().Truncate(0)
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, t2, easee.Double, "0.0"))

	//expect observation timestamp updated, value however not
	assert.Equal(t, t2, e.obsTime[easee.SESSION_ENERGY])
	assert.Equal(t, float64(20), e.sessionEnergy)
}

func TestProductUpdate_LifetimeEnergyAndSessionStartEnergy(t *testing.T) {
	e := newEasee()

	assert.False(t, e.optionalStatePresent())

	now := time.Now().UTC().Truncate(0) //truncate removes sub nanos

	tc := []struct {
		obsId           easee.ObservationID
		dataType        easee.DataType
		value           string
		expectInitState bool
	}{
		{easee.TOTAL_POWER, easee.Double, "11.0", false},
		{easee.SESSION_ENERGY, easee.Double, "22.0", false},
		{easee.LIFETIME_ENERGY, easee.Double, "1000.0", true},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		e.ProductUpdate(createPayload(tc.obsId, now, tc.dataType, tc.value))
		assert.Equal(t, tc.expectInitState, e.optionalStatePresent())
	}
}

// TestInExpectedOpMode tests the inExpectedOpMode function with different scenarios
func TestInExpectedOpMode(t *testing.T) {
	tc := []struct {
		opMode int
		enable bool
		expect bool
	}{
		{easee.ModeDisconnected, false, false},
		{easee.ModeAwaitingAuthentication, false, true},
		{easee.ModeAwaitingStart, false, true},
		{easee.ModeOffline, false, false},

		//enable cases
		{easee.ModeAwaitingAuthentication, true, false},
		{easee.ModeOffline, true, false},
		{easee.ModeCharging, true, true},
		{easee.ModeCompleted, true, true},
		{easee.ModeAwaitingStart, true, true},
		{easee.ModeReadyToCharge, true, true},
	}
	for _, tc := range tc {
		t.Logf("%+v", tc)

		e := newEasee()
		e.opMode = tc.opMode
		res := e.inExpectedOpMode(tc.enable)
		assert.Equal(t, tc.expect, res)
	}
}

func TestEasee_waitForTickResponse(t *testing.T) {
	testCases := []struct {
		name         string
		expectedTick int64
		cmdCValue    *easee.SignalRCommandResponse
		expectedErr  error
	}{
		{
			name:         "Success - Tick Found",
			expectedTick: 123,
			cmdCValue:    &easee.SignalRCommandResponse{Ticks: 123, WasAccepted: true},
			expectedErr:  nil,
		},
		{
			name:         "Success - Tick Found, but Rejected",
			expectedTick: 456,
			cmdCValue:    &easee.SignalRCommandResponse{Ticks: 456, WasAccepted: false},
			expectedErr:  fmt.Errorf("command rejected: %d", 456),
		},
		{
			name:         "Timeout",
			expectedTick: 789,
			expectedErr:  api.ErrTimeout,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("%+v", tc)

			e := newEasee()

			ch := make(chan easee.SignalRCommandResponse, 1)
			if tc.cmdCValue != nil {
				ch <- *tc.cmdCValue
			}

			err := e.waitForTickResponse(ch)

			// Assert the result
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEasee_postJsonAndWait(t *testing.T) {
	const chargerID string = "TESTTEST"
	const ticks int64 = 638798974487432600

	settingsUri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, chargerID)
	commandUri := fmt.Sprintf("%s/chargers/%s/commands/resume_charging", easee.API, chargerID)

	settingsReply := fmt.Sprintf("{\"device\":\"%s\",\"commandId\":48,\"ticks\":%d}", chargerID, ticks)

	cmdResponse := easee.SignalRCommandResponse{
		WasAccepted: true,
		Ticks:       ticks,
	}

	testCases := []struct {
		uri      string
		httpRc   int
		respBody string
		cmdResp  *easee.SignalRCommandResponse
		noop     bool
		err      error
	}{
		{settingsUri, 200, "", nil, false, nil},                                   //sync reply
		{settingsUri, 202, "[]", nil, true, nil},                                  //noop reply
		{settingsUri, 202, "[" + settingsReply + "]", nil, false, api.ErrTimeout}, //timeout
		{commandUri, 202, "{}", nil, true, nil},                                   //noop command reply
		{commandUri, 202, settingsReply, &cmdResponse, false, nil},                //command reply
		{commandUri, 400, "", nil, false, fmt.Errorf("invalid status: %d", 400)},  //unexpected result
	}

	for _, tc := range testCases {
		t.Logf("%+v", tc)

		e := newEasee()

		httpmock.ActivateNonDefault(e.Client)
		httpmock.RegisterResponder(http.MethodPost, tc.uri,
			httpmock.NewStringResponder(tc.httpRc, tc.respBody))

		if tc.cmdResp != nil {
			go func() {
				// wait for postJSONAndWait to register the per-tick channel
				var ch chan easee.SignalRCommandResponse
				for {
					e.cmdMu.Lock()
					ch = e.pendingTicks[tc.cmdResp.Ticks]
					e.cmdMu.Unlock()
					if ch != nil {
						break
					}
					time.Sleep(time.Millisecond)
				}
				ch <- *tc.cmdResp
			}()
		}

		noop, err := e.postJSONAndWait(tc.uri, nil)

		assert.Equal(t, tc.noop, noop)
		assert.Equal(t, tc.err, err)

		httpmock.Reset()
	}
}

func TestEasee_waitForChargerEnabledState(t *testing.T) {
	testCases := []struct {
		expEnabled  bool
		updateState bool
		sendObs     bool
		expectErr   error
	}{
		{false, false, false, nil},           // short circuit, already in target state
		{true, true, true, nil},              // normal flow
		{true, false, false, api.ErrTimeout}, // missing state change
		{true, true, false, nil},             // late landing state change (transition without Obs)
	}

	for _, tc := range testCases {
		t.Logf("%+v", tc)

		e := newEasee()
		e.opMode = easee.ModeAwaitingAuthentication

		if tc.updateState { // simulate state changes
			go func() {
				e.opMode = easee.ModeCharging // transition to charging
				if tc.sendObs {
					e.obsC <- easee.Observation{
						ID: easee.CHARGER_OP_MODE,
					}
				}
			}()
		}

		err := e.waitForChargerEnabledState(tc.expEnabled)

		if tc.expectErr != nil {
			assert.EqualError(t, err, tc.expectErr.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestEasee_waitForDynamicChargerCurrent(t *testing.T) {
	testCases := []struct {
		expectedDcc float64
		updateState bool
		sendObs     bool
		expectErr   error
	}{
		{6, false, false, nil},             // short circuit, already at target dcc
		{32, true, true, nil},              // normal flow
		{32, false, false, api.ErrTimeout}, // missing state change
		{32, true, false, nil},             // late landing state change (transition without Obs)
	}

	for _, tc := range testCases {
		t.Logf("%+v", tc)

		e := newEasee()
		e.dynamicChargerCurrent = 6

		if tc.updateState { // simulate state changes
			go func() {
				e.dynamicChargerCurrent = 32 // transition to 32A
				if tc.sendObs {
					e.obsC <- easee.Observation{
						ID:       easee.DYNAMIC_CHARGER_CURRENT,
						DataType: easee.Double,
						Value:    "32",
					}
				}
			}()
		}

		err := e.waitForDynamicChargerCurrent(tc.expectedDcc)

		if tc.expectErr != nil {
			assert.EqualError(t, err, tc.expectErr.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestEasee_StatusReason(t *testing.T) {
	testcases := []struct {
		opMode         int
		expectedReason api.Reason
	}{
		{easee.ModeAwaitingAuthentication, api.ReasonWaitingForAuthorization},
		{easee.ModeCompleted, api.ReasonDisconnectRequired},
		{easee.ModeDisconnected, api.ReasonUnknown},
		{easee.ModeAwaitingStart, api.ReasonUnknown},
		{easee.ModeCharging, api.ReasonUnknown},
		{easee.ModeReadyToCharge, api.ReasonUnknown},
		{easee.ModeDeauthenticating, api.ReasonUnknown},
	}
	for _, tc := range testcases {
		t.Logf("%+v", tc)

		e := newEasee()
		e.opMode = tc.opMode

		reason, err := e.StatusReason()
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedReason, reason)
	}
}

func TestEasee_MaxCurrent(t *testing.T) {
	testCases := []struct {
		targetCurrent int64
		expectCurrent float64
	}{
		{6, 6},   // short circuit
		{32, 16}, // target above max
		{10, 10}, // normal case
	}

	for _, tc := range testCases {
		t.Logf("%+v", tc)

		e := newEasee()
		e.charger = "CHARGERID"
		e.maxChargerCurrent = 16
		e.dynamicChargerCurrent = 6

		uriPattern := fmt.Sprintf("=~%s.*", easee.API)

		//register mock NoOp reply, suffices for this test case
		httpmock.ActivateNonDefault(e.Client)
		httpmock.RegisterResponder(http.MethodPost, uriPattern,
			httpmock.NewStringResponder(202, "[]"))

		err := e.MaxCurrent(tc.targetCurrent)

		assert.NoError(t, err)
		assert.Equal(t, tc.expectCurrent, e.current)
		//TODO this fails, either current or dynamicChargerCurrent need to go
		//assert.Equal(t, e.current, e.dynamicChargerCurrent)
	}
}

func TestEasee_CommandResponse_rogue(t *testing.T) {
	e := newEasee()

	rogueResp := easee.SignalRCommandResponse{
		SerialNumber: "EH123456",
		Ticks:        999999999,
		WasAccepted:  true,
		ResultCode:   0,
	}

	raw, err := json.Marshal(rogueResp)
	require.NoError(t, err)

	// No pending tick registered → should log WARN (not panic, not block)
	assert.NotPanics(t, func() {
		e.CommandResponse(raw)
	})

	// pendingTicks should still be empty
	e.cmdMu.Lock()
	assert.Empty(t, e.pendingTicks)
	e.cmdMu.Unlock()
}

func TestEasee_CommandResponse_legitimate(t *testing.T) {
	e := newEasee()

	ticks := int64(638798974487432600)
	ch := make(chan easee.SignalRCommandResponse, 1)
	e.registerPendingTick(ticks, ch)

	resp := easee.SignalRCommandResponse{
		SerialNumber: "EH123456",
		Ticks:        ticks,
		WasAccepted:  true,
	}

	raw, err := json.Marshal(resp)
	require.NoError(t, err)

	e.CommandResponse(raw)

	// Channel should have received the response
	select {
	case got := <-ch:
		assert.Equal(t, ticks, got.Ticks)
		assert.True(t, got.WasAccepted)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("CommandResponse did not deliver to pending channel")
	}
}

func TestEasee_CommandResponse_expectedOrphan(t *testing.T) {
	e := newEasee()

	// Pre-register the expected orphan
	e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

	resp := easee.SignalRCommandResponse{
		SerialNumber: "EH123456",
		ID:           int(easee.CIRCUIT_MAX_CURRENT_P1),
		Ticks:        111111111,
		WasAccepted:  true,
		ResultCode:   0,
	}

	raw, err := json.Marshal(resp)
	require.NoError(t, err)

	// Should not panic and should consume the orphan counter
	assert.NotPanics(t, func() {
		e.CommandResponse(raw)
	})

	// Counter should now be zero — a second response would be rogue
	assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
}

func TestEasee_CommandResponse_rogueAfterOrphanConsumed(t *testing.T) {
	e := newEasee()

	// Register and immediately consume via CommandResponse
	e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

	resp := easee.SignalRCommandResponse{
		SerialNumber: "EH123456",
		ID:           int(easee.CIRCUIT_MAX_CURRENT_P1),
		Ticks:        111111111,
		WasAccepted:  true,
	}
	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	e.CommandResponse(raw) // consumes the counter

	// A second identical response with counter=0 should be treated as rogue (not panic)
	assert.NotPanics(t, func() {
		e.CommandResponse(raw)
	})

	// pendingTicks untouched
	e.cmdMu.Lock()
	assert.Empty(t, e.pendingTicks)
	e.cmdMu.Unlock()
}

func TestEasee_CommandResponse_matchedByID(t *testing.T) {
	e := newEasee()

	ch := make(chan easee.SignalRCommandResponse, 1)
	e.registerPendingByID(easee.LOCATION, ch)
	defer e.unregisterPendingByID(easee.LOCATION)

	// Ticks do NOT match any pendingTicks entry — only the ID matches
	resp := easee.SignalRCommandResponse{
		SerialNumber: "EH123456",
		ID:           int(easee.LOCATION),
		Ticks:        999999999, // not in pendingTicks
		WasAccepted:  true,
		ResultCode:   0,
	}

	raw, err := json.Marshal(resp)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		e.CommandResponse(raw)
	})

	select {
	case got := <-ch:
		assert.Equal(t, resp.Ticks, got.Ticks)
		assert.True(t, got.WasAccepted)
	default:
		t.Fatal("expected CommandResponse to be delivered to pendingByID channel")
	}

	// pendingByID consumed the response — expectedOrphans untouched
	assert.False(t, e.consumeExpectedOrphan(easee.LOCATION))
}

func TestEasee_registerAndConsumeExpectedOrphan(t *testing.T) {
	e := newEasee()

	// Not registered yet — consume returns false
	assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))

	// Register once
	e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

	// First consume succeeds
	assert.True(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))

	// Second consume fails (counter back to zero)
	assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
}

func TestEasee_registerExpectedOrphan_multipleRegistrations(t *testing.T) {
	e := newEasee()

	// Register twice (two concurrent calls in flight)
	e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)
	e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

	assert.True(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
	assert.True(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
	assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
}

func TestEasee_Phases1p3p_registersExpectedOrphan(t *testing.T) {
	const siteID = 12345
	const circuitID = 67890
	const chargerID = "TESTTEST"

	e := newEasee()
	e.charger = chargerID
	e.site = siteID
	e.circuit = circuitID

	httpmock.ActivateNonDefault(e.Client)
	defer httpmock.DeactivateAndReset()

	// Mock GET circuit settings
	getURI := fmt.Sprintf("%s/sites/%d/circuits/%d/settings", easee.API, siteID, circuitID)
	maxP1, maxP2, maxP3 := 32.0, 32.0, 32.0
	getResp := easee.CircuitSettings{
		MaxCircuitCurrentP1: &maxP1,
		MaxCircuitCurrentP2: &maxP2,
		MaxCircuitCurrentP3: &maxP3,
	}
	body, err := json.Marshal(getResp)
	require.NoError(t, err)
	httpmock.RegisterResponder(http.MethodGet, getURI,
		httpmock.NewBytesResponder(200, body))

	// Mock POST circuit settings — return 200 (sync)
	httpmock.RegisterResponder(http.MethodPost, getURI,
		httpmock.NewStringResponder(200, ""))

	err = e.Phases1p3p(1)
	assert.NoError(t, err)

	// The orphan counter should have been registered before the POST.
	// Since no CommandResponse arrived in this test, the counter stays at 1.
	e.cmdMu.Lock()
	count := e.expectedOrphans[easee.CIRCUIT_MAX_CURRENT_P1]
	e.cmdMu.Unlock()
	assert.Equal(t, 1, count, "expected orphan should be registered before the POST")
}
