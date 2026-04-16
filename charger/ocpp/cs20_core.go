package ocpp

import (
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/authorization"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/meter"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/security"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// provisioning.CSMSHandler

func (cs *CSMS20) OnBootNotification(chargingStationId string, request *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	cp, err := cs.ChargingStationByID(chargingStationId)
	if err != nil {
		return nil, err
	}

	cs.log.DEBUG.Printf("charge station %s: boot notification: %+v", chargingStationId, request)

	cp.mu.Lock()
	cp.stopBootTimer()
	cp.BootNotificationResult = request
	cp.mu.Unlock()

	// try to send to channel, drop if full
	select {
	case cp.bootNotificationRequestC <- request:
	default:
	}

	// mark as connected
	cp.connect(true)

	return &provisioning.BootNotificationResponse{
		CurrentTime: types.NewDateTime(time.Now()),
		Interval:    60,
		Status:      provisioning.RegistrationStatusAccepted,
	}, nil
}

func (cs *CSMS20) OnNotifyReport(chargingStationId string, request *provisioning.NotifyReportRequest) (*provisioning.NotifyReportResponse, error) {
	cs.log.TRACE.Printf("charge station %s: notify report: %+v", chargingStationId, request)
	return &provisioning.NotifyReportResponse{}, nil
}

// authorization.CSMSHandler

func (cs *CSMS20) OnAuthorize(chargingStationId string, request *authorization.AuthorizeRequest) (*authorization.AuthorizeResponse, error) {
	cs.log.TRACE.Printf("charge station %s: authorize: %+v", chargingStationId, request)

	return &authorization.AuthorizeResponse{
		IdTokenInfo: types.IdTokenInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

// availability.CSMSHandler

func (cs *CSMS20) OnHeartbeat(chargingStationId string, request *availability.HeartbeatRequest) (*availability.HeartbeatResponse, error) {
	cs.log.TRACE.Printf("charge station %s: heartbeat", chargingStationId)

	return &availability.HeartbeatResponse{
		CurrentTime: *types.NewDateTime(time.Now()),
	}, nil
}

func (cs *CSMS20) OnStatusNotification(chargingStationId string, request *availability.StatusNotificationRequest) (*availability.StatusNotificationResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: status notification: evse=%d connector=%d status=%s",
		chargingStationId, request.EvseID, request.ConnectorID, request.ConnectorStatus)

	// cache status in registration
	cs.mu.Lock()
	if reg, ok := cs.regs[chargingStationId]; ok {
		reg.mu.Lock()
		reg.status[request.EvseID] = request
		reg.mu.Unlock()
	}
	cs.mu.Unlock()

	// forward to EVSE
	if cp, err := cs.ChargingStationByID(chargingStationId); err == nil {
		if evse := cp.evseByID(request.EvseID); evse != nil {
			evse.OnStatusNotification(request)
		}
	}

	return &availability.StatusNotificationResponse{}, nil
}

// transactions.CSMSHandler

func (cs *CSMS20) OnTransactionEvent(chargingStationId string, request *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: transaction event: type=%s trigger=%s txn=%s evse=%d seqNo=%d",
		chargingStationId, request.EventType, request.TriggerReason, request.TransactionInfo.TransactionID,
		func() int {
			if request.Evse != nil {
				return request.Evse.ID
			}
			return 0
		}(), request.SequenceNo)

	cp, err := cs.ChargingStationByID(chargingStationId)
	if err != nil {
		return nil, err
	}

	// determine EVSE
	var evse *EVSE
	if request.Evse != nil {
		evse = cp.evseByID(request.Evse.ID)
	}
	if evse == nil {
		// try to find by transaction ID
		evse = cp.evseByTransactionID(request.TransactionInfo.TransactionID)
	}

	if evse != nil {
		evse.OnTransactionEvent(request)
	}

	return &transactions.TransactionEventResponse{}, nil
}

// meter.CSMSHandler

func (cs *CSMS20) OnMeterValues(chargingStationId string, request *meter.MeterValuesRequest) (*meter.MeterValuesResponse, error) {
	cs.log.TRACE.Printf("charge station %s: meter values: evse=%d", chargingStationId, request.EvseID)

	cp, err := cs.ChargingStationByID(chargingStationId)
	if err != nil {
		return nil, err
	}

	if evse := cp.evseByID(request.EvseID); evse != nil {
		evse.updateMeterValues(request.MeterValue)
	}

	return &meter.MeterValuesResponse{}, nil
}

// security.CSMSHandler

func (cs *CSMS20) OnSecurityEventNotification(chargingStationId string, request *security.SecurityEventNotificationRequest) (*security.SecurityEventNotificationResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: security event: %s", chargingStationId, request.Type)
	return &security.SecurityEventNotificationResponse{}, nil
}

func (cs *CSMS20) OnSignCertificate(chargingStationId string, request *security.SignCertificateRequest) (*security.SignCertificateResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: sign certificate request", chargingStationId)
	return &security.SignCertificateResponse{
		Status: types.GenericStatusRejected,
	}, nil
}

// smartcharging.CSMSHandler

func (cs *CSMS20) OnClearedChargingLimit(chargingStationId string, request *smartcharging.ClearedChargingLimitRequest) (*smartcharging.ClearedChargingLimitResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: cleared charging limit", chargingStationId)
	return &smartcharging.ClearedChargingLimitResponse{}, nil
}

func (cs *CSMS20) OnNotifyChargingLimit(chargingStationId string, request *smartcharging.NotifyChargingLimitRequest) (*smartcharging.NotifyChargingLimitResponse, error) {
	cs.log.TRACE.Printf("charge station %s: notify charging limit", chargingStationId)
	return &smartcharging.NotifyChargingLimitResponse{}, nil
}

func (cs *CSMS20) OnNotifyEVChargingNeeds(chargingStationId string, request *smartcharging.NotifyEVChargingNeedsRequest) (*smartcharging.NotifyEVChargingNeedsResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: notify EV charging needs: evse=%d", chargingStationId, request.EvseID)
	return &smartcharging.NotifyEVChargingNeedsResponse{
		Status: smartcharging.EVChargingNeedsStatusAccepted,
	}, nil
}

func (cs *CSMS20) OnNotifyEVChargingSchedule(chargingStationId string, request *smartcharging.NotifyEVChargingScheduleRequest) (*smartcharging.NotifyEVChargingScheduleResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: notify EV charging schedule: evse=%d", chargingStationId, request.EvseID)
	return &smartcharging.NotifyEVChargingScheduleResponse{
		Status: types.GenericStatusAccepted,
	}, nil
}

func (cs *CSMS20) OnReportChargingProfiles(chargingStationId string, request *smartcharging.ReportChargingProfilesRequest) (*smartcharging.ReportChargingProfilesResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: report charging profiles: evse=%d profiles=%d",
		chargingStationId, request.EvseID, len(request.ChargingProfile))
	return &smartcharging.ReportChargingProfilesResponse{}, nil
}
