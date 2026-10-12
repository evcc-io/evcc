package ocpp2

import (
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/authorization"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/diagnostics"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/meter"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// ── provisioning.CSMSHandler ──────────────────────────────────────────────────

func (cs *CS2) OnBootNotification(id string, req *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	if st, err := cs.StationByID(id); err == nil {
		return st.OnBootNotification(req)
	}
	return &provisioning.BootNotificationResponse{
		CurrentTime: types.NewDateTime(time.Now()),
		Interval:    int(Timeout.Seconds()),
		Status:      provisioning.RegistrationStatusPending,
	}, nil
}

func (cs *CS2) OnNotifyReport(id string, req *provisioning.NotifyReportRequest) (*provisioning.NotifyReportResponse, error) {
	cs.log.DEBUG.Printf("%s: NotifyReport requestId=%d seqNo=%d", id, req.RequestID, req.SeqNo)
	return new(provisioning.NotifyReportResponse), nil
}

// ── availability.CSMSHandler ──────────────────────────────────────────────────

func (cs *CS2) OnHeartbeat(id string, req *availability.HeartbeatRequest) (*availability.HeartbeatResponse, error) {
	return &availability.HeartbeatResponse{
		CurrentTime: *types.NewDateTime(time.Now()),
	}, nil
}

func (cs *CS2) OnStatusNotification(id string, req *availability.StatusNotificationRequest) (*availability.StatusNotificationResponse, error) {
	cs.mu.Lock()
	if reg, ok := cs.regs[id]; ok && req != nil {
		reg.mu.Lock()
		reg.status[req.EvseID] = req
		reg.mu.Unlock()
	}
	cs.mu.Unlock()

	if st, err := cs.StationByID(id); err == nil {
		return st.OnStatusNotification(req)
	}
	return new(availability.StatusNotificationResponse), nil
}

// ── transactions.CSMSHandler ──────────────────────────────────────────────────

func (cs *CS2) OnTransactionEvent(id string, req *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
	if st, err := cs.StationByID(id); err == nil {
		return st.OnTransactionEvent(req)
	}
	return &transactions.TransactionEventResponse{
		IDTokenInfo: &types.IdTokenInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

// ── meter.CSMSHandler ─────────────────────────────────────────────────────────

func (cs *CS2) OnMeterValues(id string, req *meter.MeterValuesRequest) (*meter.MeterValuesResponse, error) {
	if st, err := cs.StationByID(id); err == nil {
		return st.OnMeterValues(req)
	}
	return new(meter.MeterValuesResponse), nil
}

// ── authorization.CSMSHandler ─────────────────────────────────────────────────

func (cs *CS2) OnAuthorize(id string, req *authorization.AuthorizeRequest) (*authorization.AuthorizeResponse, error) {
	cs.log.DEBUG.Printf("%s: Authorize IdToken=%s Type=%s", id, req.IdToken.IdToken, req.IdToken.Type)
	return &authorization.AuthorizeResponse{
		IdTokenInfo: types.IdTokenInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

// ── smartcharging.CSMSHandler ─────────────────────────────────────────────────

func (cs *CS2) OnReportChargingProfiles(id string, req *smartcharging.ReportChargingProfilesRequest) (*smartcharging.ReportChargingProfilesResponse, error) {
	cs.log.DEBUG.Printf("%s: ReportChargingProfiles requestId=%d", id, req.RequestID)
	return new(smartcharging.ReportChargingProfilesResponse), nil
}

func (cs *CS2) OnNotifyChargingLimit(id string, req *smartcharging.NotifyChargingLimitRequest) (*smartcharging.NotifyChargingLimitResponse, error) {
	cs.log.DEBUG.Printf("%s: NotifyChargingLimit evseId=%d", id, req.EvseID)
	return new(smartcharging.NotifyChargingLimitResponse), nil
}

func (cs *CS2) OnNotifyEVChargingSchedule(id string, req *smartcharging.NotifyEVChargingScheduleRequest) (*smartcharging.NotifyEVChargingScheduleResponse, error) {
	cs.log.DEBUG.Printf("%s: NotifyEVChargingSchedule evseId=%d", id, req.EvseID)
	return new(smartcharging.NotifyEVChargingScheduleResponse), nil
}

func (cs *CS2) OnNotifyEVChargingNeeds(id string, req *smartcharging.NotifyEVChargingNeedsRequest) (*smartcharging.NotifyEVChargingNeedsResponse, error) {
	cs.log.DEBUG.Printf("%s: NotifyEVChargingNeeds evseId=%d", id, req.EvseID)
	return new(smartcharging.NotifyEVChargingNeedsResponse), nil
}

func (cs *CS2) OnClearedChargingLimit(id string, req *smartcharging.ClearedChargingLimitRequest) (*smartcharging.ClearedChargingLimitResponse, error) {
	cs.log.DEBUG.Printf("%s: ClearedChargingLimit", id)
	return new(smartcharging.ClearedChargingLimitResponse), nil
}

// ── diagnostics.CSMSHandler ───────────────────────────────────────────────────

func (cs *CS2) OnNotifyEvent(id string, req *diagnostics.NotifyEventRequest) (*diagnostics.NotifyEventResponse, error) {
	return new(diagnostics.NotifyEventResponse), nil
}

func (cs *CS2) OnNotifyMonitoringReport(id string, req *diagnostics.NotifyMonitoringReportRequest) (*diagnostics.NotifyMonitoringReportResponse, error) {
	return new(diagnostics.NotifyMonitoringReportResponse), nil
}

func (cs *CS2) OnNotifyCustomerInformation(id string, req *diagnostics.NotifyCustomerInformationRequest) (*diagnostics.NotifyCustomerInformationResponse, error) {
	return new(diagnostics.NotifyCustomerInformationResponse), nil
}

func (cs *CS2) OnLogStatusNotification(id string, req *diagnostics.LogStatusNotificationRequest) (*diagnostics.LogStatusNotificationResponse, error) {
	return new(diagnostics.LogStatusNotificationResponse), nil
}

// ── firmware.CSMSHandler ──────────────────────────────────────────────────────

func (cs *CS2) OnFirmwareStatusNotification(id string, req *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationResponse, error) {
	cs.log.DEBUG.Printf("%s: FirmwareStatus: %s", id, req.Status)
	return new(firmware.FirmwareStatusNotificationResponse), nil
}

func (cs *CS2) OnPublishFirmwareStatusNotification(id string, req *firmware.PublishFirmwareStatusNotificationRequest) (*firmware.PublishFirmwareStatusNotificationResponse, error) {
	return new(firmware.PublishFirmwareStatusNotificationResponse), nil
}