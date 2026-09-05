package charger

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOCPPChargingProfile(t *testing.T) {
	for _, tc := range []struct {
		name        string
		txProfile   bool
		txID        int
		relative    bool
		zeroStack   bool
		wantTxID    int
		wantPurpose types.ChargingProfilePurposeType
	}{
		{"default idle", false, 0, false, false, 0, types.ChargingProfilePurposeTxDefaultProfile},
		{"default active", false, 42, false, false, 0, types.ChargingProfilePurposeTxDefaultProfile},
		{"transaction idle", true, 0, false, false, 0, types.ChargingProfilePurposeTxDefaultProfile},
		{"transaction active", true, 42, false, false, 42, types.ChargingProfilePurposeTxProfile},
		{"relative transaction", true, 42, true, true, 42, types.ChargingProfilePurposeTxProfile},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &OCPP{
				cp:        &ocpp.CP{ChargingProfileId: 3, StackLevel: 5, ChargingRateUnit: types.ChargingRateUnitAmperes},
				txProfile: tc.txProfile, profileKindRelative: tc.relative, stackLevelZero: tc.zeroStack,
			}
			profile := c.createChargingProfile(7.2, tc.txID)
			assert.Equal(t, tc.wantPurpose, profile.ChargingProfilePurpose)
			assert.Equal(t, tc.wantTxID, profile.TransactionId)
			assert.Equal(t, 3, profile.ChargingProfileId)
			assert.Equal(t, 7.2, profile.ChargingSchedule.ChargingSchedulePeriod[0].Limit)
			if tc.relative {
				assert.Equal(t, types.ChargingProfileKindRelative, profile.ChargingProfileKind)
				assert.Nil(t, profile.ChargingSchedule.StartSchedule)
			} else {
				assert.Equal(t, types.ChargingProfileKindAbsolute, profile.ChargingProfileKind)
				require.NotNil(t, profile.ChargingSchedule.StartSchedule)
				assert.True(t, profile.ChargingSchedule.StartSchedule.Before(time.Now()))
			}
			if tc.zeroStack {
				assert.Zero(t, profile.StackLevel)
			} else {
				assert.Equal(t, 5, profile.StackLevel)
			}
			data, err := json.Marshal(profile)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(data, &fields))
			_, hasTransactionID := fields["transactionId"]
			assert.Equal(t, tc.wantTxID != 0, hasTransactionID)
		})
	}
}

func TestOCPPChargingProfileValidation(t *testing.T) {
	for _, value := range []string{"", "unknown", "ChargePointMaxProfile"} {
		_, err := NewOCPPFromConfig(t.Context(), map[string]any{"chargingprofile": value})
		require.ErrorContains(t, err, "invalid charging profile")
	}
}

func TestOCPPChargingProfileTemplates(t *testing.T) {
	for _, name := range []string{"ocpp", "ocpp-autoaid"} {
		for _, value := range []string{"", "TxDefaultProfile", "TxProfile"} {
			t.Run(name+"/"+value, func(t *testing.T) {
				tmpl, err := templates.ByName(templates.Charger, name)
				require.NoError(t, err)
				params := map[string]any{}
				if value != "" {
					params["chargingprofile"] = value
				}
				rendered, _, err := tmpl.RenderResult(templates.Charger, templates.RenderModeInstance, params)
				require.NoError(t, err)
				if value == "TxProfile" {
					assert.Contains(t, string(rendered), "chargingprofile: TxProfile")
				} else {
					assert.NotContains(t, string(rendered), "chargingprofile:")
				}
			})
		}
	}
}

type ocppProfileHandler struct {
	ChargePointHandler
	profiles chan *smartcharging.SetChargingProfileRequest
	reject   atomic.Bool
}

func (h *ocppProfileHandler) OnSetChargingProfile(req *smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileConfirmation, error) {
	h.profiles <- req
	status := smartcharging.ChargingProfileStatusAccepted
	if h.reject.Swap(false) {
		status = smartcharging.ChargingProfileStatusRejected
	}
	return smartcharging.NewSetChargingProfileConfirmation(status), nil
}

func (suite *ocppTestSuite) TestTransactionProfiles() {
	t := suite.T()
	old := sponsor.Subject
	sponsor.Subject = "test"
	t.Cleanup(func() { sponsor.Subject = old })

	id := suite.stationID("tx-profile")
	cp, _, stop := suite.startChargePoint(id, 1)
	h := &ocppProfileHandler{profiles: make(chan *smartcharging.SetChargingProfileRequest, 16)}
	cp.SetSmartChargingHandler(h)
	require.NoError(t, cp.Start(ocppTestUrl))
	charger, err := NewOCPPFromConfig(t.Context(), map[string]any{
		"stationid": id, "chargingprofile": "TxProfile", "connecttimeout": ocppTestConnectTimeout.String(),
	})
	require.NoError(t, err)
	c := charger.(*OCPP)

	expectProfile := func(purpose types.ChargingProfilePurposeType, txID int, current float64) {
		t.Helper()
		select {
		case req := <-h.profiles:
			assert.Equal(t, 1, req.ConnectorId)
			assert.Equal(t, purpose, req.ChargingProfile.ChargingProfilePurpose)
			assert.Equal(t, txID, req.ChargingProfile.TransactionId)
			assert.Equal(t, current, req.ChargingProfile.ChargingSchedule.ChargingSchedulePeriod[0].Limit)
		case <-time.After(time.Second):
			t.Fatal("missing charging profile")
		}
	}

	for _, status := range []core.ChargePointStatus{core.ChargePointStatusCharging, core.ChargePointStatusSuspendedEV, core.ChargePointStatusSuspendedEVSE} {
		_, err = cp.StatusNotification(1, core.NoError, status)
		require.NoError(t, err)
		require.ErrorIs(t, c.MaxCurrentMillis(7.25), api.ErrNotAvailable)
		require.Empty(t, h.profiles)
	}
	recoveredID := 77
	_, err = cp.MeterValues(1, []types.MeterValue{{
		Timestamp:    types.NewDateTime(time.Now()),
		SampledValue: []types.SampledValue{{Measurand: types.MeasurandCurrentImport, Value: "0"}},
	}}, func(req *core.MeterValuesRequest) { req.TransactionId = &recoveredID })
	require.NoError(t, err)
	_, err = c.Status()
	require.NoError(t, err)
	expectProfile(types.ChargingProfilePurposeTxProfile, recoveredID, 0)
	_, err = cp.StopTransaction(0, types.NewDateTime(time.Now()), recoveredID)
	require.NoError(t, err)
	_, err = cp.StatusNotification(1, core.NoError, core.ChargePointStatusPreparing)
	require.NoError(t, err)
	_, err = c.Status()
	require.NoError(t, err)
	expectProfile(types.ChargingProfilePurposeTxDefaultProfile, 0, 0)
	require.NoError(t, c.MaxCurrentMillis(7.25))
	expectProfile(types.ChargingProfilePurposeTxDefaultProfile, 0, 7.2)

	start, err := cp.StartTransaction(1, "tag", 0, types.NewDateTime(time.Now()))
	require.NoError(t, err)
	_, err = c.Status()
	require.NoError(t, err)
	expectProfile(types.ChargingProfilePurposeTxProfile, start.TransactionId, 7.2)
	_, err = c.Status()
	require.NoError(t, err)
	require.Empty(t, h.profiles)

	require.NoError(t, c.Enable(false))
	expectProfile(types.ChargingProfilePurposeTxProfile, start.TransactionId, 0)
	_, err = cp.StopTransaction(0, types.NewDateTime(time.Now()), start.TransactionId)
	require.NoError(t, err)
	_, err = cp.StatusNotification(1, core.NoError, core.ChargePointStatusFinishing)
	require.NoError(t, err)
	_, err = c.Status()
	require.NoError(t, err)
	expectProfile(types.ChargingProfilePurposeTxDefaultProfile, 0, 0)

	start, err = cp.StartTransaction(1, "tag", 0, types.NewDateTime(time.Now()))
	require.NoError(t, err)
	h.reject.Store(true)
	_, err = c.Status()
	require.ErrorContains(t, err, "Rejected")
	expectProfile(types.ChargingProfilePurposeTxProfile, start.TransactionId, 0)
	_, err = c.Status()
	require.NoError(t, err)
	expectProfile(types.ChargingProfilePurposeTxProfile, start.TransactionId, 0)
	require.NoError(t, c.Enable(true))
	expectProfile(types.ChargingProfilePurposeTxProfile, start.TransactionId, 7.2)

	stop()
	require.Eventually(t, func() bool { return !c.cp.Connected() }, time.Second, 10*time.Millisecond)
	require.ErrorIs(t, c.MaxCurrentMillis(10), api.ErrTimeout)
}

func (suite *ocppTestSuite) TestDefaultProfiles() {
	t := suite.T()
	old := sponsor.Subject
	sponsor.Subject = "test"
	t.Cleanup(func() { sponsor.Subject = old })
	for _, value := range []string{"", "TxDefaultProfile"} {
		id := suite.stationID("default-profile-" + value)
		cp, _, _ := suite.startChargePoint(id, 1)
		h := &ocppProfileHandler{profiles: make(chan *smartcharging.SetChargingProfileRequest, 16)}
		cp.SetSmartChargingHandler(h)
		require.NoError(t, cp.Start(ocppTestUrl))
		config := map[string]any{"stationid": id, "connecttimeout": ocppTestConnectTimeout.String()}
		if value != "" {
			config["chargingprofile"] = value
		}
		charger, err := NewOCPPFromConfig(t.Context(), config)
		require.NoError(t, err)
		_, err = cp.StartTransaction(1, "tag", 0, types.NewDateTime(time.Now()))
		require.NoError(t, err)
		_, err = charger.Status()
		require.NoError(t, err)
		require.Empty(t, h.profiles)
		require.NoError(t, charger.MaxCurrent(8))
		require.Len(t, h.profiles, 1)
		profile := (<-h.profiles).ChargingProfile
		assert.Equal(t, types.ChargingProfilePurposeTxDefaultProfile, profile.ChargingProfilePurpose)
		assert.Zero(t, profile.TransactionId)
	}
}
