package charger

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/stretchr/testify/require"
)

type huaweiScheduleHandler struct {
	*ocppProfileHandler
	events chan string
	limits chan func() float64
}

func (h *huaweiScheduleHandler) OnRemoteStartTransaction(req *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	h.events <- "start"
	return h.ChargePointHandler.OnRemoteStartTransaction(req)
}

func (h *huaweiScheduleHandler) OnGetCompositeSchedule(req *smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	h.events <- "schedule"
	select {
	case limit := <-h.limits:
		return &smartcharging.GetCompositeScheduleConfirmation{
			Status:        smartcharging.GetCompositeScheduleStatusAccepted,
			ConnectorId:   &req.ConnectorId,
			ScheduleStart: types.NewDateTime(time.Now().Truncate(time.Second)),
			ChargingSchedule: &types.ChargingSchedule{
				ChargingRateUnit:       types.ChargingRateUnitWatts,
				ChargingSchedulePeriod: []types.ChargingSchedulePeriod{types.NewChargingSchedulePeriod(0, limit())},
			},
		}, nil
	default:
		return smartcharging.NewGetCompositeScheduleConfirmation(smartcharging.GetCompositeScheduleStatusRejected), nil
	}
}

func (suite *ocppTestSuite) TestHuaweiStartAfterProfile() {
	t := suite.T()
	sponsor.SetSubjectForTest(t, "test")
	id := suite.stationID("huawei-profile")
	cp, _, _ := suite.startChargePoint(id, 1)
	h := &huaweiScheduleHandler{
		ocppProfileHandler: &ocppProfileHandler{profiles: make(chan *smartcharging.SetChargingProfileRequest, 8)},
		events:             make(chan string, 8),
		limits:             make(chan func() float64, 8),
	}
	cp.SetCoreHandler(h)
	cp.SetSmartChargingHandler(h)
	require.NoError(t, cp.Start(ocppTestUrl))
	instance, err := templates.RenderInstance(templates.Charger, map[string]any{
		"template": "ocpp-huawei", "stationid": id, "remotestart": true, "remotestartrecovery": true,
		"connecttimeout": ocppTestConnectTimeout.String(),
	})
	require.NoError(t, err)
	instance.Other["nochangeavailability"] = true
	charger, err := NewOCPPFromConfig(t.Context(), instance.Other)
	require.NoError(t, err)
	c := charger.(*OCPP)
	require.False(t, api.HasCap[api.PhaseSwitcher](c))

	status := func(value core.ChargePointStatus) {
		t.Helper()
		_, err := cp.StatusNotification(1, core.NoError, value)
		require.NoError(t, err)
	}
	expectProfile := func(watts float64) {
		t.Helper()
		select {
		case req := <-h.profiles:
			schedule := req.ChargingProfile.ChargingSchedule
			require.Equal(t, types.ChargingRateUnitWatts, schedule.ChargingRateUnit)
			require.Equal(t, watts, schedule.ChargingSchedulePeriod[0].Limit)
		case <-time.After(time.Second):
			t.Fatal("missing charging profile")
		}
	}
	expectEvent := func(event string) {
		t.Helper()
		select {
		case got := <-h.events:
			require.Equal(t, event, got)
		case <-time.After(10 * time.Second):
			t.Fatalf("missing %s", event)
		}
	}
	startTransaction := func() int {
		t.Helper()
		reply, err := cp.StartTransaction(1, "test", 0, types.NewDateTime(time.Now()))
		require.NoError(t, err)
		status(core.ChargePointStatusCharging)
		return reply.TransactionId
	}

	status(core.ChargePointStatusAvailable)
	status(core.ChargePointStatusPreparing)
	h.limits <- func() float64 { return 0 }
	h.limits <- func() float64 { return 11000 }
	now := time.Now()
	require.NoError(t, c.MaxCurrent(16))
	expectProfile(11040)
	expectEvent("schedule")
	require.GreaterOrEqual(t, time.Since(now), 7*time.Second)
	expectEvent("schedule")
	expectEvent("start")
	tx := startTransaction()

	require.NoError(t, c.Enable(false))
	expectProfile(0)
	status(core.ChargePointStatusPreparing)
	h.limits <- func() float64 { return 11000 }
	h.limits <- func() float64 { return 4830 }
	require.NoError(t, c.MaxCurrent(7))
	expectProfile(4830)
	require.Empty(t, h.events)
	_, err = cp.StopTransaction(0, types.NewDateTime(time.Now()), tx)
	require.NoError(t, err)
	expectEvent("schedule")
	expectEvent("schedule")
	expectEvent("start")
	tx = startTransaction()

	require.NoError(t, c.Enable(false))
	expectProfile(0)
	_, err = cp.StopTransaction(0, types.NewDateTime(time.Now()), tx)
	require.NoError(t, err)
	status(core.ChargePointStatusPreparing)
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	h.limits <- func() float64 { <-release; return 4830 }
	require.NoError(t, c.MaxCurrent(7))
	expectProfile(4830)
	expectEvent("schedule")
	off := make(chan error, 1)
	go func() { off <- c.Enable(false) }()
	require.Eventually(t, func() bool {
		enabled, known := c.conn.PendingStartProfile()
		return known && !enabled
	}, time.Second, 10*time.Millisecond)
	close(release)
	require.NoError(t, <-off)
	expectProfile(0)
	select {
	case got := <-h.events:
		t.Fatalf("unexpected %s after Off", got)
	case <-time.After(250 * time.Millisecond):
	}
}
