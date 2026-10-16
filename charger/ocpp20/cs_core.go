package ocpp20

import (
	"time"

	"github.com/evcc-io/evcc/charger/ocpp"
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

func (cs *CSMS) OnBootNotification(chargingStationId string, request *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	cp, err := cs.ChargingStationByID(chargingStationId)
	if err != nil {
		// unknown charging station: keep it pending until configured (parity with 1.6)
		return &provisioning.BootNotificationResponse{
			CurrentTime: types.NewDateTime(time.Now()),
			Interval:    int(ocpp.Timeout.Seconds()),
			Status:      provisioning.RegistrationStatusPending,
		}, nil
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

func (cs *CSMS) OnNotifyReport(chargingStationId string, request *provisioning.NotifyReportRequest) (*provisioning.NotifyReportResponse, error) {
	cs.log.TRACE.Printf("charge station %s: notify report: %+v", chargingStationId, request)
	return &provisioning.NotifyReportResponse{}, nil
}

// authorization.CSMSHandler

func (cs *CSMS) OnAuthorize(chargingStationId string, request *authorization.AuthorizeRequest) (*authorization.AuthorizeResponse, error) {
	cs.log.TRACE.Printf("charge station %s: authorize: %+v", chargingStationId, request)

	return &authorization.AuthorizeResponse{
		IdTokenInfo: types.IdTokenInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

// availability.CSMSHandler

func (cs *CSMS) OnHeartbeat(chargingStationId string, request *availability.HeartbeatRequest) (*availability.HeartbeatResponse, error) {
	cs.log.TRACE.Printf("charge station %s: heartbeat", chargingStationId)

	return &availability.HeartbeatResponse{
		CurrentTime: *types.NewDateTime(time.Now()),
	}, nil
}

func (cs *CSMS) OnStatusNotification(chargingStationId string, request *availability.StatusNotificationRequest) (*availability.StatusNotificationResponse, error) {
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

	// fan-out to all matching EVSE listeners (EVSE-wide and matching connector)
	if cp, err := cs.ChargingStationByID(chargingStationId); err == nil {
		for _, evse := range cp.evsesForConnector(request.EvseID, request.ConnectorID) {
			evse.OnStatusNotification(request)
		}
	}

	return &availability.StatusNotificationResponse{}, nil
}

// transactions.CSMSHandler

func (cs *CSMS) OnTransactionEvent(chargingStationId string, request *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
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

	// fan-out by EVSE (and connector when present); fall back to txn-id lookup
	if request.Evse != nil {
		connectorID := 0
		if request.Evse.ConnectorID != nil {
			connectorID = *request.Evse.ConnectorID
		}
		matched := cp.evsesForConnector(request.Evse.ID, connectorID)
		for _, evse := range matched {
			evse.OnTransactionEvent(request)
		}
		if len(matched) == 0 {
			if evse := cp.evseByTransactionID(request.TransactionInfo.TransactionID); evse != nil {
				evse.OnTransactionEvent(request)
			}
		}
	} else if evse := cp.evseByTransactionID(request.TransactionInfo.TransactionID); evse != nil {
		evse.OnTransactionEvent(request)
	}

	return &transactions.TransactionEventResponse{}, nil
}

// meter.CSMSHandler

func (cs *CSMS) OnMeterValues(chargingStationId string, request *meter.MeterValuesRequest) (*meter.MeterValuesResponse, error) {
	cs.log.TRACE.Printf("charge station %s: meter values: evse=%d", chargingStationId, request.EvseID)

	cp, err := cs.ChargingStationByID(chargingStationId)
	if err != nil {
		return nil, err
	}

	// MeterValues has no connector field — fan out to every EVSE listener
	// registered for this EVSE id (multi-connector consumers see EVSE-wide samples).
	for _, evse := range cp.evsesForEvse(request.EvseID) {
		evse.updateMeterValues(request.MeterValue)
	}

	return &meter.MeterValuesResponse{}, nil
}

// security.CSMSHandler

func (cs *CSMS) OnSecurityEventNotification(chargingStationId string, request *security.SecurityEventNotificationRequest) (*security.SecurityEventNotificationResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: security event: %s", chargingStationId, request.Type)
	return &security.SecurityEventNotificationResponse{}, nil
}

func (cs *CSMS) OnSignCertificate(chargingStationId string, request *security.SignCertificateRequest) (*security.SignCertificateResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: sign certificate request", chargingStationId)
	return &security.SignCertificateResponse{
		Status: types.GenericStatusRejected,
	}, nil
}

// smartcharging.CSMSHandler

func (cs *CSMS) OnClearedChargingLimit(chargingStationId string, request *smartcharging.ClearedChargingLimitRequest) (*smartcharging.ClearedChargingLimitResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: cleared charging limit", chargingStationId)
	return &smartcharging.ClearedChargingLimitResponse{}, nil
}

func (cs *CSMS) OnNotifyChargingLimit(chargingStationId string, request *smartcharging.NotifyChargingLimitRequest) (*smartcharging.NotifyChargingLimitResponse, error) {
	cs.log.TRACE.Printf("charge station %s: notify charging limit", chargingStationId)
	return &smartcharging.NotifyChargingLimitResponse{}, nil
}

func (cs *CSMS) OnNotifyEVChargingNeeds(chargingStationId string, request *smartcharging.NotifyEVChargingNeedsRequest) (*smartcharging.NotifyEVChargingNeedsResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: notify EV charging needs: evse=%d", chargingStationId, request.EvseID)
	return &smartcharging.NotifyEVChargingNeedsResponse{
		Status: smartcharging.EVChargingNeedsStatusAccepted,
	}, nil
}

func (cs *CSMS) OnNotifyEVChargingSchedule(chargingStationId string, request *smartcharging.NotifyEVChargingScheduleRequest) (*smartcharging.NotifyEVChargingScheduleResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: notify EV charging schedule: evse=%d", chargingStationId, request.EvseID)
	return &smartcharging.NotifyEVChargingScheduleResponse{
		Status: types.GenericStatusAccepted,
	}, nil
}

func (cs *CSMS) OnReportChargingProfiles(chargingStationId string, request *smartcharging.ReportChargingProfilesRequest) (*smartcharging.ReportChargingProfilesResponse, error) {
	cs.log.DEBUG.Printf("charge station %s: report charging profiles: evse=%d profiles=%d",
		chargingStationId, request.EvseID, len(request.ChargingProfile))
	return &smartcharging.ReportChargingProfilesResponse{}, nil
}
