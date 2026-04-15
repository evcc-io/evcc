package charger

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/mocks"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ghostone"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestGhostEEBusREST creates a GhostEEBus with a minimal EEBus (no EV connected), for pure REST tests.
func newTestGhostEEBusREST(t *testing.T) *GhostEEBus {
	t.Helper()
	return &GhostEEBus{
		EEBus: &EEBus{
			log: util.NewLogger("test"),
			cem: &eebus.CustomerEnergyManagement{},
		},
		Helper: request.NewHelper(util.NewLogger("test")),
		uri:    "https://wallbox.local/api/v2",
	}
}

// newTestGhostEEBusWithEEBus creates a GhostEEBus with mocked EEBUS dependencies.
func newTestGhostEEBusWithEEBus(t *testing.T) (*GhostEEBus, *mocks.CemEVCCInterface, *spinemocks.EntityRemoteInterface) {
	t.Helper()

	evccMock := mocks.NewCemEVCCInterface(t)
	evEntity := spinemocks.NewEntityRemoteInterface(t)

	eb := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC: evccMock,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	wb := &GhostEEBus{
		EEBus:  eb,
		Helper: request.NewHelper(util.NewLogger("test")),
		uri:    "https://wallbox.local/api/v2",
	}

	return wb, evccMock, evEntity
}

const ghostEEBusRelaisStateURL = "https://wallbox.local/api/v2/system/relais-switch/state"

func TestGhostEEBus_PhaseSwitchISO15118(t *testing.T) {
	tests := []struct {
		name        string
		connected   bool
		comStandard model.DeviceConfigurationKeyValueStringType
		comErr      error
		wantErr     error
	}{
		{
			name:      "no_ev_connected",
			connected: false,
		},
		{
			name:        "iec61851_allowed",
			connected:   true,
			comStandard: model.DeviceConfigurationKeyValueStringTypeIEC61851,
		},
		{
			name:        "iso15118_ed1_blocked",
			connected:   true,
			comStandard: model.DeviceConfigurationKeyValueStringTypeISO151182ED1,
			wantErr:     api.ErrNotAvailable,
		},
		{
			name:        "iso15118_ed2_blocked",
			connected:   true,
			comStandard: model.DeviceConfigurationKeyValueStringTypeISO151182ED2,
			wantErr:     api.ErrNotAvailable,
		},
		{
			name:      "com_standard_error",
			connected: true,
			comErr:    errors.New("data not available"),
			wantErr:   api.ErrNotAvailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, evccMock, evEntity := newTestGhostEEBusWithEEBus(t)

			httpmock.ActivateNonDefault(wb.Client)
			defer httpmock.DeactivateAndReset()

			evccMock.EXPECT().EVConnected(evEntity).Return(tc.connected)
			if tc.connected {
				if tc.comErr != nil {
					evccMock.EXPECT().CommunicationStandard(evEntity).Return(
						model.DeviceConfigurationKeyValueStringType(""), tc.comErr,
					)
				} else {
					evccMock.EXPECT().CommunicationStandard(evEntity).Return(tc.comStandard, nil)
				}
			}

			if tc.wantErr == nil {
				// mock PUT + GET for successful phase switch
				httpmock.RegisterResponder(http.MethodPut, ghostEEBusRelaisStateURL,
					httpmock.NewStringResponder(200, ""),
				)
				body, _ := json.Marshal(ghostone.RelaisSwitchStateRead{
					Value: ghostone.RelaisStateOnePhase, CurrentState: ghostone.RelaisStateOnePhase,
				})
				httpmock.RegisterResponder(http.MethodGet, ghostEEBusRelaisStateURL,
					httpmock.NewBytesResponder(200, body),
				)
			}

			err := wb.phases1p3p(1)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGhostEEBus_PhaseSwitch(t *testing.T) {
	tests := []struct {
		name      string
		phases    int
		wantValue string
		readBack  ghostone.RelaisSwitchStateRead
		wantErr   bool
	}{
		{
			name:      "switch_to_1p",
			phases:    1,
			wantValue: ghostone.RelaisStateOnePhase,
			readBack:  ghostone.RelaisSwitchStateRead{Value: ghostone.RelaisStateSwitchInProgress, CurrentState: ghostone.RelaisStateThreePhase},
		},
		{
			name:      "switch_to_3p",
			phases:    3,
			wantValue: ghostone.RelaisStateThreePhase,
			readBack:  ghostone.RelaisSwitchStateRead{Value: ghostone.RelaisStateSwitchInProgress, CurrentState: ghostone.RelaisStateOnePhase},
		},
		{
			name:      "already_in_target_state",
			phases:    1,
			wantValue: ghostone.RelaisStateOnePhase,
			readBack:  ghostone.RelaisSwitchStateRead{Value: ghostone.RelaisStateOnePhase, CurrentState: ghostone.RelaisStateOnePhase},
		},
		{
			name:      "denied_not_possible",
			phases:    1,
			wantValue: ghostone.RelaisStateOnePhase,
			readBack:  ghostone.RelaisSwitchStateRead{Value: ghostone.RelaisStateNotPossible, LimitationReason: "hlcLimitation"},
			wantErr:   true,
		},
		{
			name:      "denied_not_available",
			phases:    3,
			wantValue: ghostone.RelaisStateThreePhase,
			readBack:  ghostone.RelaisSwitchStateRead{Value: ghostone.RelaisStateNotAvailable},
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb := newTestGhostEEBusREST(t)

			httpmock.ActivateNonDefault(wb.Client)
			defer httpmock.DeactivateAndReset()

			// mock PUT (phase switch command)
			var capturedBody ghostone.RelaisSwitchStateWrite
			httpmock.RegisterResponder(http.MethodPut, ghostEEBusRelaisStateURL,
				func(req *http.Request) (*http.Response, error) {
					if err := json.NewDecoder(req.Body).Decode(&capturedBody); err != nil {
						return httpmock.NewStringResponse(400, ""), nil
					}
					return httpmock.NewStringResponse(200, ""), nil
				},
			)

			// mock GET (read-after-write verification)
			body, _ := json.Marshal(tc.readBack)
			httpmock.RegisterResponder(http.MethodGet, ghostEEBusRelaisStateURL,
				httpmock.NewBytesResponder(200, body),
			)

			err := wb.phases1p3p(tc.phases)

			assert.Equal(t, tc.wantValue, capturedBody.Value)
			assert.Equal(t, 2, httpmock.GetTotalCallCount(), "expected PUT + GET")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGhostEEBus_GetPhases(t *testing.T) {
	tests := []struct {
		name         string
		currentState string
		value        string
		wantPhases   int
		wantErr      bool
	}{
		{
			name:         "one_phase",
			currentState: ghostone.RelaisStateOnePhase,
			value:        ghostone.RelaisStateOnePhase,
			wantPhases:   1,
		},
		{
			name:         "three_phase",
			currentState: ghostone.RelaisStateThreePhase,
			value:        ghostone.RelaisStateThreePhase,
			wantPhases:   3,
		},
		{
			name:         "switch_in_progress",
			currentState: ghostone.RelaisStateOnePhase,
			value:        ghostone.RelaisStateSwitchInProgress,
			wantErr:      true,
		},
		{
			name:         "unknown_state",
			currentState: "foo",
			value:        "foo",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb := newTestGhostEEBusREST(t)

			httpmock.ActivateNonDefault(wb.Client)
			defer httpmock.DeactivateAndReset()

			resp := ghostone.RelaisSwitchStateRead{
				Value:        tc.value,
				CurrentState: tc.currentState,
			}
			body, _ := json.Marshal(resp)
			httpmock.RegisterResponder(http.MethodGet, ghostEEBusRelaisStateURL,
				httpmock.NewBytesResponder(200, body),
			)

			phases, err := wb.getPhases()

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantPhases, phases)
			}
		})
	}
}

func TestGhostEEBus_Decorator(t *testing.T) {
	wb := newTestGhostEEBusREST(t)

	// with phase switching
	decorated := decorateGhostEEBus(wb, nil, nil, nil, wb.phases1p3p, wb.getPhases)
	assert.True(t, api.HasCap[api.PhaseSwitcher](decorated), "expected PhaseSwitcher")
	assert.True(t, api.HasCap[api.PhaseGetter](decorated), "expected PhaseGetter")

	// without phase switching
	decorated = decorateGhostEEBus(wb, nil, nil, nil, nil, nil)
	assert.False(t, api.HasCap[api.PhaseSwitcher](decorated), "unexpected PhaseSwitcher")
}

func TestGhostEEBus_Config(t *testing.T) {
	// test that config decoding works (fails on missing eebus instance, not on decoding)
	ctx := context.Background()
	config := map[string]any{
		"ski":      "test-ski",
		"ip":       "10.0.1.30",
		"user":     "technician",
		"password": "secret",
	}

	_, err := NewGhostEEBusFromConfig(ctx, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eebus not configured")
}

func TestGhostEEBus_ConfigMissingCredentials(t *testing.T) {
	// without credentials, falls back to pure EEBUS (no REST features)
	// fails on missing eebus instance, not on credentials
	ctx := context.Background()
	config := map[string]any{
		"ski":   "test-ski",
		"ip":    "10.0.1.30",
		"meter": true,
	}

	_, err := NewGhostEEBusFromConfig(ctx, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eebus not configured")
}

func TestGhostEEBus_Identify(t *testing.T) {
	tests := []struct {
		name   string
		uuid   string
		wantID string
	}{
		{
			name:   "with_uuid",
			uuid:   "ABC123",
			wantID: "ABC123",
		},
		{
			name:   "empty",
			uuid:   "",
			wantID: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb := newTestGhostEEBusREST(t)

			httpmock.ActivateNonDefault(wb.Client)
			defer httpmock.DeactivateAndReset()

			resp := ghostone.RfidCardLastRead{UUID: tc.uuid}
			body, _ := json.Marshal(resp)
			httpmock.RegisterResponder(http.MethodGet,
				"https://wallbox.local/api/v2/rfid-cards/last-read",
				httpmock.NewBytesResponder(200, body),
			)

			id, err := wb.identify()
			require.NoError(t, err)
			assert.Equal(t, tc.wantID, id)
		})
	}
}

func TestGhostEEBus_IdentifyFallback(t *testing.T) {
	t.Run("rfid_disabled_falls_back_to_eebus", func(t *testing.T) {
		wb, evccMock, evEntity := newTestGhostEEBusWithEEBus(t)
		wb.hasRFID = false

		// EEBus identification returns MAC
		evccMock.EXPECT().EVConnected(evEntity).Return(true)
		evccMock.EXPECT().Identifications(evEntity).Return([]ucapi.IdentificationItem{{Value: "MAC-001"}}, nil)

		id, err := wb.Identify()
		require.NoError(t, err)
		assert.Equal(t, "MAC-001", id)
	})

	t.Run("rfid_returns_id", func(t *testing.T) {
		wb, _, _ := newTestGhostEEBusWithEEBus(t)
		wb.hasRFID = true

		httpmock.ActivateNonDefault(wb.Client)
		defer httpmock.DeactivateAndReset()

		resp := ghostone.RfidCardLastRead{UUID: "RFID-42"}
		body, _ := json.Marshal(resp)
		httpmock.RegisterResponder(http.MethodGet,
			"https://wallbox.local/api/v2/rfid-cards/last-read",
			httpmock.NewBytesResponder(200, body),
		)

		id, err := wb.Identify()
		require.NoError(t, err)
		assert.Equal(t, "RFID-42", id)
	})

	t.Run("rfid_empty_falls_back_to_eebus", func(t *testing.T) {
		wb, evccMock, evEntity := newTestGhostEEBusWithEEBus(t)
		wb.hasRFID = true

		httpmock.ActivateNonDefault(wb.Client)
		defer httpmock.DeactivateAndReset()

		// RFID returns empty
		resp := ghostone.RfidCardLastRead{UUID: ""}
		body, _ := json.Marshal(resp)
		httpmock.RegisterResponder(http.MethodGet,
			"https://wallbox.local/api/v2/rfid-cards/last-read",
			httpmock.NewBytesResponder(200, body),
		)

		// Falls back to EEBus
		evccMock.EXPECT().EVConnected(evEntity).Return(true)
		evccMock.EXPECT().Identifications(evEntity).Return([]ucapi.IdentificationItem{{Value: "MAC-001"}}, nil)

		id, err := wb.Identify()
		require.NoError(t, err)
		assert.Equal(t, "MAC-001", id)
	})
}
