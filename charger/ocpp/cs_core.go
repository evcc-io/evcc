package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
)

// cs actions

func (cs *CS) TriggerResetRequest(cp *CP, resetType core.ResetType) {
	if err := cs.Reset(cp.id, func(request *core.ResetConfirmation, err error) {
		log := cs.log.TRACE
		if err == nil && request != nil && request.Status != core.ResetStatusAccepted {
			log = cs.log.ERROR
		}

		var status core.ResetStatus
		if request != nil {
			status = request.Status
		}

		log.Printf("TriggerReset for %s: %+v", cp.id, status)
	}, resetType); err != nil {
		cs.log.ERROR.Printf("send TriggerReset for %s failed: %v", cp.id, err)
	}
}

func (cs *CS) TriggerMeterValuesRequest(cp *CP) {
	if err := cs.TriggerMessage(cp.id, func(request *remotetrigger.TriggerMessageConfirmation, err error) {
		log := cs.log.TRACE
		if err == nil && request != nil && request.Status != remotetrigger.TriggerMessageStatusAccepted {
			log = cs.log.ERROR
		}

		var status remotetrigger.TriggerMessageStatus
		if request != nil {
			status = request.Status
		}

		log.Printf("TriggerMessage %s for %s: %+v", core.MeterValuesFeatureName, cp.id, status)
	}, core.MeterValuesFeatureName); err != nil {
		cs.log.ERROR.Printf("send TriggerMessage for %s failed: %v", cp.id, err)
	}
}

// cp actions

func (cs *CS) OnAuthorize(chargePointId string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.Authorize(request)
}

func (cs *CS) OnBootNotification(chargePointId string, request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.BootNotification(request)
}

func (cs *CS) OnDataTransfer(chargePointId string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.DataTransfer(request)
}

func (cs *CS) OnHeartbeat(chargePointId string, request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.Heartbeat(request)
}

func (cs *CS) OnMeterValues(chargePointId string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.MeterValues(request)
}

func (cs *CS) OnStatusNotification(chargePointId string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StatusNotification(request)
}

func (cs *CS) OnStartTransaction(chargePointId string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StartTransaction(request)
}

func (cs *CS) OnStopTransaction(chargePointId string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StopTransaction(request)
}

func (cs *CS) OnDiagnosticsStatusNotification(chargePointId string, request *firmware.DiagnosticsStatusNotificationRequest) (confirmation *firmware.DiagnosticsStatusNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.DiagnosticStatusNotification(request)
}

func (cs *CS) OnFirmwareStatusNotification(chargePointId string, request *firmware.FirmwareStatusNotificationRequest) (confirmation *firmware.FirmwareStatusNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.FirmwareStatusNotification(request)
}
