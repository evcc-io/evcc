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
		Helper:    request.NewHelper(log),
		obsTime:   make(map[easee.ObservationID]time.Time),
		log:       log,
		startDone: func() {},
		cmdC:      make(chan easee.SignalRCommandResponse),
		obsC:      make(chan easee.Observation),
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

func TestProductUpdate_InitialStateCheck(t *testing.T) {
	now := time.Now().UTC().Truncate(0) //truncate removes sub nanos

	e := newEasee()

	assert.False(t, e.optionalStatePresent())

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

			// Set up the command channel for test and Easee to share
			cmdC := make(chan easee.SignalRCommandResponse, 1) // make it buffered for ease of testing
			e.cmdC = cmdC

			if tc.cmdCValue != nil {
				cmdC <- *tc.cmdCValue
			}

			err := e.waitForTickResponse(tc.expectedTick)

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
				e.cmdC <- *tc.cmdResp
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
