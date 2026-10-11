package charger

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jarcoal/httpmock"
	"github.com/philippseith/signalr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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
				// opMode is read under the lock by inExpectedOpMode
				e.mux.Lock()
				e.opMode = easee.ModeCharging // transition to charging
				e.mux.Unlock()
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
				// dynamicChargerCurrent is read under the lock by waitForDynamicChargerCurrent
				e.mux.Lock()
				e.dynamicChargerCurrent = 32 // transition to 32A
				e.mux.Unlock()
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

	_, err = e.dispatcher.Send(uri, nil)
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

	_, err = e.dispatcher.Send(uri, nil)
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

func TestChargeSessionStart_SetsFields(t *testing.T) {
	e := newEasee()
	e.energyMeter.Initialized = true
	e.totalEnergy = 9100.0

	data := easee.ChargingSessionStartData{ID: 801, Start: time.Now(), MeterValue: 9141.414622}
	jsonData, _ := json.Marshal(data)
	e.ProductUpdate(createPayload(easee.CHARGE_SESSION_START, time.Now(), easee.String, string(jsonData)))

	assert.Equal(t, 801, e.currentSessionID)

	total, err := e.TotalEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 9141.414622, total)
}

func TestChargingSession_UpdatesBothWhenIdMatches(t *testing.T) {
	e := newEasee()
	e.energyMeter.Initialized = true
	e.currentSessionID = 801
	e.totalEnergy = 9100.0

	data := easee.ChargingSessionData{ID: 801, Start: time.Now().Add(-time.Hour), Stop: time.Now(), EnergyKwh: 16.2, MeterValueStart: 9157.3, MeterValueStop: 9173.5}
	jsonData, _ := json.Marshal(data)
	e.ProductUpdate(createPayload(easee.CHARGING_SESSION, time.Now(), easee.String, string(jsonData)))

	charged, err := e.ChargedEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 16.2, charged)

	total, err := e.TotalEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 9173.5, total)
}

func TestChargingSession_MismatchedId_ProtectsSessionEnergy(t *testing.T) {
	e := newEasee()
	e.energyMeter.Initialized = true
	e.currentSessionID = 100
	e.sessionEnergy = 5.0
	e.totalEnergy = 9150.0

	data := easee.ChargingSessionData{ID: 99, Start: time.Now().Add(-time.Hour), Stop: time.Now(), EnergyKwh: 7.5, MeterValueStart: 9166, MeterValueStop: 9173.5}
	jsonData, _ := json.Marshal(data)
	e.ProductUpdate(createPayload(easee.CHARGING_SESSION, time.Now(), easee.String, string(jsonData)))

	charged, err := e.ChargedEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 5.0, charged) // sessionEnergy unchanged

	total, err := e.TotalEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 9173.5, total) // totalEnergy updated
}

func TestChargingSession_AtStartup_ProtectsSessionEnergy(t *testing.T) {
	e := newEasee()
	now := time.Now().UTC()
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, now.Add(-time.Second), easee.Integer, "1"))
	// currentSessionId is 0 by default
	e.sessionEnergy = 0
	e.totalEnergy = 9150.0

	data := easee.ChargingSessionData{ID: 803, Start: now.Add(-time.Hour), Stop: now, EnergyKwh: 19.08, MeterValueStart: 9154.42, MeterValueStop: 9173.5}
	jsonData, _ := json.Marshal(data)
	e.ProductUpdate(createPayload(easee.CHARGING_SESSION, time.Now(), easee.String, string(jsonData)))

	charged, err := e.ChargedEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 0.0, charged) // sessionEnergy protected (Id 803 != 0)

	total, err := e.TotalEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 9173.5, total) // totalEnergy updated
}

func TestLifetimeEnergy_DoesNotDecreaseTotalEnergy(t *testing.T) {
	e := newEasee()
	e.energyMeter.Initialized = true
	e.totalEnergy = 9173.5

	e.ProductUpdate(createPayload(easee.LIFETIME_ENERGY, time.Now(), easee.Double, "9170.0"))

	total, err := e.TotalEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 9173.5, total)
}

func TestChargerOpMode_ConnectResetsSessionFields(t *testing.T) {
	e := newEasee()
	e.opMode = easee.ModeDisconnected
	e.currentSessionID = 803
	e.sessionEnergy = 5.0

	// Transition from disconnected to awaiting start
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, time.Now(), easee.Integer, fmt.Sprintf("%d", easee.ModeAwaitingStart)))

	assert.Equal(t, 0, e.currentSessionID)

	charged, err := e.ChargedEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 0.0, charged)
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

func TestEasee_Enable_clampsCurrentOnChargeStart(t *testing.T) {
	const chargerID = "TESTTEST"

	tests := []struct {
		name        string
		current     float64
		expectClamp bool
	}{
		{"6A clamped to 7A", 6, true},
		{"7A unchanged", 7, false},
		{"10A unchanged", 10, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEasee()
			e.charger = chargerID
			e.current = tc.current
			e.dynamicChargerCurrent = tc.current
			e.opMode = easee.ModeAwaitingAuthentication
			e.authorize = true
			e.chargerEnabled = true // skip the "enable charger once" path

			httpmock.ActivateNonDefault(e.Client)
			defer httpmock.DeactivateAndReset()

			var callOrder []string

			// Mock POST charger settings (DCC clamp)
			settingsURI := fmt.Sprintf("%s/chargers/%s/settings", easee.API, chargerID)
			httpmock.RegisterResponder(http.MethodPost, settingsURI, func(req *http.Request) (*http.Response, error) {
				callOrder = append(callOrder, "settings")
				return httpmock.NewStringResponse(202, "[]"), nil
			})

			// Mock POST start_charging command
			startURI := fmt.Sprintf("%s/chargers/%s/commands/%s", easee.API, chargerID, easee.ChargeStart)
			httpmock.RegisterResponder(http.MethodPost, startURI, func(req *http.Request) (*http.Response, error) {
				callOrder = append(callOrder, "start")
				// Transition to expected state for waitForChargerEnabledState(true)
				e.mux.Lock()
				e.opMode = easee.ModeAwaitingStart
				e.mux.Unlock()
				return httpmock.NewStringResponse(200, ""), nil
			})

			err := e.Enable(true)
			assert.NoError(t, err)

			if tc.expectClamp {
				assert.Equal(t, []string{"settings", "start"}, callOrder,
					"DCC clamp should be sent before ChargeStart")
				assert.Equal(t, 7.0, e.current, "current should be clamped to 7")
			} else {
				assert.Equal(t, []string{"start"}, callOrder,
					"no DCC clamp should be sent")
				assert.Equal(t, tc.current, e.current, "current should be unchanged")
			}
		})
	}
}

func TestEaseeEnergyStartupObservations(t *testing.T) {
	start := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	anchor := easee.ChargingSessionStartData{ID: 42, Start: start, MeterValue: 10000}
	stop := easee.ChargingSessionData{ID: 42, Start: start, Stop: start.Add(time.Hour), MeterValueStart: 10000, MeterValueStop: 10008, EnergyKwh: 8}
	for _, test := range []struct {
		name      string
		status    int
		empty     bool
		active    bool
		wantError bool
	}{
		{name: "completion"},
		{name: "fetch fails", status: http.StatusServiceUnavailable, wantError: true},
		{name: "empty batch", empty: true, wantError: true},
		{name: "newer live energy", active: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := newEasee()
			e.charger = "TEST"
			e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, anchor))
			e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(time.Hour), easee.Double, "9"))
			e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(time.Hour), easee.Integer, "4"))
			if test.active {
				e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(2*time.Hour), easee.Integer, "3"))
				e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(2*time.Hour), easee.Double, "10"))
			}
			anchorJSON, err := json.Marshal(anchor)
			require.NoError(t, err)
			stopJSON, err := json.Marshal(stop)
			require.NoError(t, err)
			body := map[string]any{"observations": []any{
				map[string]any{"id": 223, "timestamp": start, "dataType": 6, "value": string(anchorJSON)},
				map[string]any{"id": 121, "timestamp": start.Add(time.Hour), "dataType": 3, "value": 9},
				map[string]any{"id": 129, "timestamp": start.Add(time.Hour), "dataType": 6, "value": string(stopJSON)},
			}}
			if test.empty {
				body["observations"] = []any{}
			}
			if test.active {
				body["observations"] = []any{
					map[string]any{"id": 223, "timestamp": start, "dataType": 6, "value": string(anchorJSON)},
					map[string]any{"id": 121, "timestamp": start.Add(time.Hour), "dataType": 3, "value": 8},
					map[string]any{"id": 109, "timestamp": start.Add(time.Hour), "dataType": 4, "value": 2},
				}
			}
			status := test.status
			if status == 0 {
				status = http.StatusOK
			}
			responder, err := httpmock.NewJsonResponder(status, body)
			require.NoError(t, err)
			httpmock.ActivateNonDefault(e.Client)
			t.Cleanup(httpmock.DeactivateAndReset)
			httpmock.RegisterResponder(http.MethodGet, "https://api.easee.com/state/TEST/observations?ids=109,121,124,129,223", responder)
			err = e.recoverEnergy(context.Background())
			if test.wantError {
				require.Error(t, err)
				assert.False(t, e.energyMeter.Initialized)
				return
			}
			require.NoError(t, err)
			total, err := e.TotalEnergy()
			require.NoError(t, err)
			want := 10008.0
			if test.active {
				want = 10010
			}
			assert.Equal(t, want, total)
		})
	}
}

func easeeSessionPayload(t *testing.T, id easee.ObservationID, ts time.Time, data any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(data)
	require.NoError(t, err)
	return createPayload(id, ts, easee.String, string(b))
}

func TestEaseeEnergyRestartReplay(t *testing.T) {
	start := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	for _, mode := range []int{easee.ModeCharging, easee.ModeAwaitingStart, easee.ModeCompleted} {
		observations := []json.RawMessage{
			easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 42, Start: start, MeterValue: 10000}),
			createPayload(easee.SESSION_ENERGY, start.Add(45*time.Minute), easee.Double, "8"),
			createPayload(easee.LIFETIME_ENERGY, start.Add(30*time.Minute), easee.Double, "10006"),
			createPayload(easee.CHARGER_OP_MODE, start.Add(40*time.Minute), easee.Integer, fmt.Sprint(mode)),
		}
		for a := range 4 {
			for b := range 4 {
				for c := range 4 {
					for d := range 4 {
						if a == b || a == c || a == d || b == c || b == d || c == d {
							continue
						}
						t.Run(fmt.Sprintf("%d/%d%d%d%d", mode, a, b, c, d), func(t *testing.T) {
							e := newEasee()
							for _, i := range []int{a, b, c, d} {
								e.ProductUpdate(observations[i])
							}
							total, err := e.TotalEnergy()
							require.NoError(t, err)
							assert.Equal(t, 10008.0, total)
							charged, err := e.ChargedEnergy()
							require.NoError(t, err)
							assert.Equal(t, 8.0, charged)
							for _, observation := range observations {
								e.ProductUpdate(observation)
							}
							total, err = e.TotalEnergy()
							require.NoError(t, err)
							assert.Equal(t, 10008.0, total)
						})
					}
				}
			}
		}
	}
}

func TestEaseeEnergyNeedsSessionAnchor(t *testing.T) {
	e := newEasee()
	now := time.Now().UTC()
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, now, easee.Integer, "3"))
	e.ProductUpdate(createPayload(easee.LIFETIME_ENERGY, now.Add(-time.Minute), easee.Double, "10006"))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, now, easee.Double, "8"))
	_, err := e.TotalEnergy()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestEaseeEnergySessionBoundary(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 1, Start: start, MeterValue: 10000}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(10*time.Minute), easee.Double, "8"))
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"))
	_, err := e.TotalEnergy()
	require.NoError(t, err)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(20*time.Minute), easee.Integer, "1"))
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(30*time.Minute), easee.Integer, "2"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start.Add(30*time.Minute), easee.ChargingSessionStartData{ID: 2, Start: start.Add(30 * time.Minute), MeterValue: 10008}))
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10008.0, total)
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(31*time.Minute), easee.Double, "2"))
	total, err = e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10010.0, total)
}

func TestEaseeEnergyRecovery(t *testing.T) {
	start := time.Now().UTC().Add(-time.Hour)
	for _, tc := range []struct {
		name         string
		observations []json.RawMessage
		ready        bool
	}{
		{"active session", []json.RawMessage{
			createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"),
			easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 42, Start: start, MeterValue: 10000}),
			createPayload(easee.SESSION_ENERGY, start.Add(time.Minute), easee.Double, "8"),
		}, true},
		{"missing anchor", []json.RawMessage{
			createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"),
			createPayload(easee.LIFETIME_ENERGY, start.Add(time.Minute), easee.Double, "10000"),
			createPayload(easee.SESSION_ENERGY, start.Add(time.Minute), easee.Double, "8"),
		}, false},
		{"missing energy", []json.RawMessage{
			createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"),
			easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 42, Start: start, MeterValue: 10000}),
		}, false},
		{"idle physical meter", []json.RawMessage{
			createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "1"),
			createPayload(easee.LIFETIME_ENERGY, start.Add(time.Minute), easee.Double, "10008"),
		}, true},
		{"stale idle meter", []json.RawMessage{
			createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "1"),
			createPayload(easee.LIFETIME_ENERGY, start.Add(-time.Minute), easee.Double, "10006"),
		}, false},
		{"completed session", []json.RawMessage{
			createPayload(easee.CHARGER_OP_MODE, start.Add(time.Minute), easee.Integer, "1"),
			easeeSessionPayload(t, easee.CHARGING_SESSION, start.Add(2*time.Minute), easee.ChargingSessionData{ID: 42, Start: start, Stop: start.Add(time.Minute), MeterValueStop: 10008, EnergyKwh: 8}),
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEasee()
			e.Client.Timeout = time.Millisecond
			for _, observation := range tc.observations {
				e.ProductUpdate(observation)
			}
			err := e.waitForEnergy(context.Background())
			if tc.ready {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
		})
	}
}

func TestEaseeEnergyCompletedSession(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(20*time.Minute), easee.Integer, "1"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGING_SESSION, start.Add(20*time.Minute), easee.ChargingSessionData{ID: 1, Start: start, Stop: start.Add(20 * time.Minute), MeterValueStart: 10000, MeterValueStop: 10008, EnergyKwh: 8}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(21*time.Minute), easee.Double, "9"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 1, Start: start, MeterValue: 10000}))
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10008.0, total)
}

func TestEaseeEnergyCompletionReplayOrder(t *testing.T) {
	start := time.Now().UTC().Add(-time.Hour)
	anchor := easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 1, Start: start, MeterValue: 10000})
	energy := createPayload(easee.SESSION_ENERGY, start.Add(21*time.Minute), easee.Double, "9")
	stop := easeeSessionPayload(t, easee.CHARGING_SESSION, start.Add(20*time.Minute), easee.ChargingSessionData{ID: 1, Start: start, Stop: start.Add(20 * time.Minute), MeterValueStop: 10008, EnergyKwh: 8})
	for _, order := range [][]json.RawMessage{
		{anchor, energy, stop}, {anchor, stop, energy},
		{energy, anchor, stop}, {energy, stop, anchor},
		{stop, anchor, energy}, {stop, energy, anchor},
	} {
		e := newEasee()
		e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(20*time.Minute), easee.Integer, "1"))
		for _, observation := range order {
			e.ProductUpdate(observation)
		}
		total, err := e.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, 10008.0, total)
		charged, err := e.ChargedEnergy()
		require.NoError(t, err)
		assert.Equal(t, 8.0, charged)
	}
}

func TestEaseeEnergyReconciliation(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 42, Start: start, MeterValue: 10000}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(time.Minute), easee.Double, "8"))
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10008.0, total)
	for i, tc := range []struct {
		id    easee.ObservationID
		value string
		want  float64
	}{
		{easee.LIFETIME_ENERGY, "10006", 10008},
		{easee.LIFETIME_ENERGY, "10008", 10008},
		{easee.SESSION_ENERGY, "0", 10008},
		{easee.SESSION_ENERGY, "7", 10008},
		{easee.SESSION_ENERGY, "9", 10009},
		{easee.LIFETIME_ENERGY, "10010", 10010},
		{easee.SESSION_ENERGY, "10", 10010},
		{easee.SESSION_ENERGY, "11", 10011},
	} {
		e.ProductUpdate(createPayload(tc.id, start.Add(time.Duration(i+2)*time.Minute), easee.Double, tc.value))
		total, err := e.TotalEnergy()
		require.NoError(t, err)
		assert.Equal(t, tc.want, total)
	}
}

func TestEaseeEnergyDelayedPreviousSession(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "1"))
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(20*time.Minute), easee.Integer, "2"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start.Add(20*time.Minute), easee.ChargingSessionStartData{ID: 2, Start: start.Add(20 * time.Minute), MeterValue: 10008}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(10*time.Minute), easee.Double, "8"))
	charged, err := e.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 0.0, charged)
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(21*time.Minute), easee.Double, "2"))
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10010.0, total)
}

func TestEaseeEnergyNewSessionBeforeMode(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "1"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start.Add(20*time.Minute), easee.ChargingSessionStartData{ID: 2, Start: start.Add(20 * time.Minute), MeterValue: 10008}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(21*time.Minute), easee.Double, "2"))
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(20*time.Minute), easee.Integer, "2"))
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10010.0, total)
	charged, err := e.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 2.0, charged)
}

func TestEaseeEnergyRecoveryCancellation(t *testing.T) {
	e := newEasee()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, e.waitForEnergy(ctx), context.Canceled)
}

func TestEaseeEnergyRecoveryZeroTimeout(t *testing.T) {
	e := newEasee()
	e.Client.Timeout = 0
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	assert.ErrorIs(t, e.waitForEnergy(ctx), context.DeadlineExceeded)
	assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
}

func TestEaseeEnergyRecoveryAfterLateReplay(t *testing.T) {
	e := newEasee()
	e.Client.Timeout = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- e.waitForEnergy(ctx) }()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(time.Minute), easee.Double, "8"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 42, Start: start, MeterValue: 10000}))
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10008.0, total)
}

func TestEaseeEnergyUnknownState(t *testing.T) {
	e := newEasee()
	_, err := e.TotalEnergy()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestEaseeEnergySessionStartsBeforeChargingMode(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "1"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start.Add(time.Minute), easee.ChargingSessionStartData{ID: 42, Start: start.Add(time.Minute), MeterValue: 10000}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(2*time.Minute), easee.Double, "8"))
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(3*time.Minute), easee.Integer, "3"))
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10008.0, total)
}

func TestEaseeEnergyMissingSessionStart(t *testing.T) {
	start := time.Now().UTC().Add(-time.Hour)
	e := newEasee()
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 42, MeterValue: 10000}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(time.Minute), easee.Double, "8"))
	_, err := e.TotalEnergy()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestEaseeEnergyOverflow(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "3"))
	e.ProductUpdate(easeeSessionPayload(t, easee.CHARGE_SESSION_START, start, easee.ChargingSessionStartData{ID: 42, Start: start, MeterValue: math.MaxFloat64}))
	e.ProductUpdate(createPayload(easee.SESSION_ENERGY, start.Add(time.Minute), easee.Double, fmt.Sprint(math.MaxFloat64)))
	_, err := e.TotalEnergy()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestEaseeEnergyRecoveryRetainsUsableMeter(t *testing.T) {
	e := newEasee()
	start := time.Now().UTC().Add(-time.Hour)
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, "1"))
	e.ProductUpdate(createPayload(easee.LIFETIME_ENERGY, start.Add(time.Minute), easee.Double, "10008"))
	require.NoError(t, e.waitForEnergy(context.Background()))
	e.ProductUpdate(createPayload(easee.CHARGER_OP_MODE, start.Add(2*time.Minute), easee.Integer, "3"))
	total, err := e.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 10008.0, total)
}

type easeeStartupHub struct {
	signalr.Hub
	observations []json.RawMessage
}

func (h *easeeStartupHub) SubscribeWithCurrentState(charger string, currentState bool) {
	for _, observation := range h.observations {
		h.Clients().Caller().Send("ProductUpdate", observation)
	}
}

type easeeStartupTransport struct {
	api    *httpmock.MockTransport
	stream *url.URL
}

func (tr easeeStartupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "streams.easee.com" {
		local := req.Clone(req.Context())
		local.URL.Scheme = tr.stream.Scheme
		local.URL.Host = tr.stream.Host
		local.Host = tr.stream.Host
		return http.DefaultTransport.RoundTrip(local)
	}
	return tr.api.RoundTrip(req)
}

func TestEaseeConstructorEnergyRecovery(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		mode       int
		anchor     bool
		wantEnergy bool
	}{
		{"active session", http.StatusOK, easee.ModeCharging, true, true},
		{"stopped while connected", http.StatusOK, easee.ModeCompleted, true, true},
		{"HTTP failure", http.StatusServiceUnavailable, easee.ModeCharging, true, false},
		{"missing anchor", http.StatusOK, easee.ModeCharging, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			start := time.Now().UTC().Add(-time.Hour)
			observations := []json.RawMessage{
				createPayload(easee.CHARGER_OP_MODE, start, easee.Integer, fmt.Sprint(tc.mode)),
				createPayload(easee.SESSION_ENERGY, start.Add(time.Minute), easee.Double, "8"),
				createPayload(easee.LIFETIME_ENERGY, start.Add(-time.Minute), easee.Double, "9990"),
				createPayload(easee.TOTAL_POWER, start, easee.Double, "0"),
			}
			hub, err := signalr.NewServer(ctx, signalr.HubFactory(func() signalr.HubInterface {
				return &easeeStartupHub{observations: observations}
			}))
			require.NoError(t, err)
			mux := http.NewServeMux()
			hub.MapHTTP(signalr.WithHTTPServeMux(mux), "/hubs/chargers")
			server := httptest.NewServer(mux)
			defer server.Close()
			streamURL, err := url.Parse(server.URL)
			require.NoError(t, err)
			mock := httpmock.NewMockTransport()
			mock.RegisterResponder("GET", easee.API+"/chargers/TEST/site", httpmock.NewStringResponder(200, `{}`))
			mock.RegisterResponder("GET", easee.API+"/chargers/TEST/config", httpmock.NewStringResponder(200, `{}`))
			snapshotStarted := make(chan struct{})
			releaseSnapshot := make(chan struct{})
			mock.RegisterResponder("GET", "https://api.easee.com/state/TEST/observations?ids=109,121,124,129,223", func(req *http.Request) (*http.Response, error) {
				close(snapshotStarted)
				select {
				case <-releaseSnapshot:
				case <-req.Context().Done():
					return nil, req.Context().Err()
				}
				items := []map[string]any{{"id": easee.SESSION_ENERGY, "timestamp": start.Add(time.Minute), "dataType": easee.Double, "value": 8}}
				if tc.anchor {
					items = append(items, map[string]any{"id": easee.CHARGE_SESSION_START, "timestamp": start, "dataType": easee.String, "value": fmt.Sprintf(`{"id":42,"start":"%s","meterValue":10000}`, start.Format(time.RFC3339Nano))})
				}
				return httpmock.NewJsonResponse(tc.status, map[string]any{"observations": items})
			})
			log := util.NewLogger("easee")
			helper := request.NewHelper(log)
			helper.Client.Transport = easeeStartupTransport{api: mock, stream: streamURL}
			type result struct {
				charger *Easee
				err     error
			}
			resultC := make(chan result, 1)
			go func() {
				c, err := newEaseeWithClient(ctx, helper, log, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}), "TEST", 500*time.Millisecond, false)
				resultC <- result{c, err}
			}()
			select {
			case <-snapshotStarted:
			case res := <-resultC:
				t.Fatalf("constructor returned before fetching energy: %v", res.err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			select {
			case res := <-resultC:
				t.Fatalf("constructor returned before complete snapshot: %v", res.err)
			default:
			}
			close(releaseSnapshot)
			select {
			case res := <-resultC:
				if tc.wantEnergy {
					require.NoError(t, res.err)
					total, err := res.charger.TotalEnergy()
					require.NoError(t, err)
					assert.Equal(t, 10008.0, total)
				} else {
					require.Error(t, res.err)
					if tc.status == http.StatusOK {
						assert.ErrorIs(t, res.err, context.DeadlineExceeded)
					}
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}
