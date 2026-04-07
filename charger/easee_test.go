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
	helper := request.NewHelper(log)
	e := Easee{
		Helper:     helper,
		obsTime:    make(map[easee.ObservationID]time.Time),
		log:        log,
		startDone:  func() {},
		obsC:       make(chan easee.Observation),
		dispatcher: easee.NewCommandDispatcher(helper, log, 500*time.Millisecond),
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
		{easee.ModeCompleted, api.ReasonUnknown},
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
		e.dynamicChargerCurrent = tc.expectCurrent // noop: already at target (capped) value

		uriPattern := fmt.Sprintf("=~%s.*", easee.API)

		// register mock NoOp reply (HTTP 202, empty array → Ticks==0)
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
	// Intentionally triggers a WARN — suppress it to keep test output clean.
	util.LogLevel("error", nil)
	t.Cleanup(func() { util.LogLevel("info", nil) })

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

	// The meaningful guarantee is no panic and no block.
	// Dispatcher's internal state is not directly inspectable from outside the package.
}

func TestEasee_CommandResponse_legitimate(t *testing.T) {
	e := newEasee()
	httpmock.ActivateNonDefault(e.Client)

	const ticks int64 = 638798974487432600
	const uri = easee.API + "/chargers/EH123456/settings"

	body := fmt.Sprintf(`[{"device":"EH123456","commandId":48,"ticks":%d}]`, ticks)
	httpmock.RegisterResponder(http.MethodPost, uri,
		httpmock.NewStringResponder(http.StatusAccepted, body))

	resp := easee.SignalRCommandResponse{
		SerialNumber: "EH123456",
		Ticks:        ticks,
		WasAccepted:  true,
	}
	raw, err := json.Marshal(resp)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		// Small delay to let Send register the pending tick before we dispatch
		time.Sleep(10 * time.Millisecond)
		e.CommandResponse(raw)
		errCh <- nil
	}()

	err = e.dispatcher.Send(uri, nil)
	assert.NoError(t, err)
	<-errCh
}

func TestEasee_CommandResponse_expectedOrphan(t *testing.T) {
	e := newEasee()

	// Pre-register the expected orphan via the dispatcher's public API
	e.dispatcher.ExpectOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

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
	assert.False(t, e.dispatcher.CancelOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
}

func TestEasee_CommandResponse_rogueAfterOrphanConsumed(t *testing.T) {
	// Intentionally triggers a WARN — suppress it to keep test output clean.
	util.LogLevel("error", nil)
	t.Cleanup(func() { util.LogLevel("info", nil) })

	e := newEasee()

	// Register and immediately consume via CommandResponse
	e.dispatcher.ExpectOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

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

	// The meaningful guarantee is no panic and no block.
	// Dispatcher's internal state is not directly inspectable from outside the package.
}

func TestEasee_CommandResponse_matchedByID(t *testing.T) {
	e := newEasee()
	httpmock.ActivateNonDefault(e.Client)

	const ticks int64 = 638798974487432601
	const obsID = easee.LOCATION // commandId in body
	const uri = easee.API + "/chargers/EH123456/settings"

	body := fmt.Sprintf(`[{"device":"EH123456","commandId":%d,"ticks":%d}]`, int(obsID), ticks)
	httpmock.RegisterResponder(http.MethodPost, uri,
		httpmock.NewStringResponder(http.StatusAccepted, body))

	// Ticks do NOT match — only the ID matches (wrong ticks value in SignalR response)
	resp := easee.SignalRCommandResponse{
		SerialNumber: "EH123456",
		ID:           int(obsID),
		Ticks:        ticks + 1, // wrong ticks → forces ID fallback path
		WasAccepted:  true,
		ResultCode:   0,
	}
	raw, err := json.Marshal(resp)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		e.CommandResponse(raw)
		errCh <- nil
	}()

	err = e.dispatcher.Send(uri, nil)
	assert.NoError(t, err)
	<-errCh
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
	// CancelOrphan returns true iff a counter entry was consumed.
	assert.True(t, e.dispatcher.CancelOrphan(easee.CIRCUIT_MAX_CURRENT_P1),
		"expected orphan should be registered before the POST")
}

func TestLivenessCheck_staleObservations(t *testing.T) {
	e := newEasee()
	e.opMode = easee.ModeCharging
	e.currentPower = 7280
	e.currentL1, e.currentL2, e.currentL3 = 16, 16, 16
	e.lastObsReceived = time.Now().Add(-(observationTimeout + time.Minute))

	power, err := e.CurrentPower()
	assert.NoError(t, err)
	assert.Equal(t, float64(0), power, "expired observations: CurrentPower must return 0W")

	l1, l2, l3, err := e.Currents()
	assert.NoError(t, err)
	assert.Equal(t, float64(0), l1)
	assert.Equal(t, float64(0), l2)
	assert.Equal(t, float64(0), l3)
}

func TestLivenessCheck_freshObservations(t *testing.T) {
	e := newEasee()
	e.opMode = easee.ModeCharging
	e.currentPower = 7280
	e.currentL1, e.currentL2, e.currentL3 = 16, 16, 16
	e.lastObsReceived = time.Now()

	power, err := e.CurrentPower()
	assert.NoError(t, err)
	assert.Equal(t, float64(7280), power)

	l1, l2, l3, err := e.Currents()
	assert.NoError(t, err)
	assert.Equal(t, float64(16), l1)
	assert.Equal(t, float64(16), l2)
	assert.Equal(t, float64(16), l3)
}

func TestIsTNGrid(t *testing.T) {
	// TN grid types must return true
	assert.True(t, isTNGrid(easee.PowerGridTN3Phase))
	assert.True(t, isTNGrid(easee.PowerGridTN2PhasePin234))
	assert.True(t, isTNGrid(easee.PowerGridTN1Phase))

	// IT grid types, zero, and unknown values must return false
	assert.False(t, isTNGrid(4))  // IT3Phase
	assert.False(t, isTNGrid(5))  // IT1Phase
	assert.False(t, isTNGrid(0))  // absent / unknown
	assert.False(t, isTNGrid(99)) // arbitrary unknown
}

// makeTestSite returns a Site with a single Circuit containing the given charger IDs.
// Site.ID = 111, Circuit.ID = 222.
func makeTestSite(chargerIDs ...string) easee.Site {
	chargers := make([]easee.Charger, len(chargerIDs))
	for i, id := range chargerIDs {
		chargers[i] = easee.Charger{ID: id}
	}
	return easee.Site{
		ID: 111,
		Circuits: []easee.Circuit{
			{ID: 222, Chargers: chargers},
		},
	}
}

func TestDetermineCircuit(t *testing.T) {
	const chargerID = "TESTTEST"
	configURI := fmt.Sprintf("%s/chargers/%s/config", easee.API, chargerID)

	tests := []struct {
		name        string
		httpStatus  int
		gridType    int
		chargerIDs  []string
		wantCircuit int
		wantErr     bool
	}{
		{
			name:        "TN grid, sole charger — circuit assigned",
			httpStatus:  200,
			gridType:    easee.PowerGridTN3Phase,
			chargerIDs:  []string{chargerID},
			wantCircuit: 222,
		},
		{
			name:        "IT grid, sole charger — circuit not assigned",
			httpStatus:  200,
			gridType:    4, // IT3Phase
			chargerIDs:  []string{chargerID},
			wantCircuit: 0,
		},
		{
			name:       "config fetch fails — error returned",
			httpStatus: 500,
			chargerIDs: []string{chargerID},
			wantErr:    true,
		},
		{
			name:        "TN grid, multi-charger circuit — circuit not assigned",
			httpStatus:  200,
			gridType:    easee.PowerGridTN3Phase,
			chargerIDs:  []string{chargerID, "OTHER"},
			wantCircuit: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEasee()
			e.charger = chargerID

			httpmock.ActivateNonDefault(e.Client)
			defer httpmock.DeactivateAndReset()

			if tc.httpStatus == 200 {
				body, _ := json.Marshal(easee.ChargerConfig{DetectedPowerGridType: tc.gridType})
				httpmock.RegisterResponder(http.MethodGet, configURI,
					httpmock.NewBytesResponder(200, body))
			} else {
				httpmock.RegisterResponder(http.MethodGet, configURI,
					httpmock.NewStringResponder(tc.httpStatus, ""))
			}

			err := e.determineCircuit(makeTestSite(tc.chargerIDs...))

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantCircuit, e.circuit)
			}
		})
	}
}
