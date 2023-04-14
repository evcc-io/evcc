package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
)

// cs actions

func (cs *CS) TriggerResetRequest(id string, resetType core.ResetType) {
	if err := cs.Reset(id, func(request *core.ResetConfirmation, err error) {
		log := cs.log.TRACE
		if err == nil && request != nil && request.Status != core.ResetStatusAccepted {
			log = cs.log.ERROR
		}

		var status core.ResetStatus
		if request != nil {
			status = request.Status
		}

		log.Printf("TriggerReset for %s: %+v", id, status)
	}, resetType); err != nil {
		cs.log.ERROR.Printf("send TriggerReset for %s failed: %v", id, err)
	}
}

func (cs *CS) TriggerMessageRequest(id string, requestedMessage remotetrigger.MessageTrigger, props ...func(request *remotetrigger.TriggerMessageRequest)) {
	if err := cs.TriggerMessage(id, func(request *remotetrigger.TriggerMessageConfirmation, err error) {
		log := cs.log.TRACE
		if err == nil && request != nil && request.Status != remotetrigger.TriggerMessageStatusAccepted {
			log = cs.log.ERROR
		}

		var status remotetrigger.TriggerMessageStatus
		if request != nil {
			status = request.Status
		}

		log.Printf("TriggerMessage %s for %s: %+v", requestedMessage, id, status)
	}, requestedMessage, props...); err != nil {
		cs.log.ERROR.Printf("send TriggerMessage %s for %s failed: %v", requestedMessage, id, err)
	}
}

func (cs *CS) TriggerMeterValuesRequest(id string, connector int) {
	cs.TriggerMessageRequest(id, core.MeterValuesFeatureName, func(request *remotetrigger.TriggerMessageRequest) {
		request.ConnectorId = &connector
	})
}

// cp actions

func (cs *CS) OnAuthorize(id string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.Authorize(request)
}

func (cs *CS) OnBootNotification(id string, request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.BootNotification(request)
}

func (cs *CS) OnDataTransfer(id string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.DataTransfer(request)
}

func (cs *CS) OnHeartbeat(id string, request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.Heartbeat(request)
}

func (cs *CS) OnMeterValues(id string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.MeterValues(request)
}

func (cs *CS) OnStatusNotification(id string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.StatusNotification(request)
}

func (cs *CS) OnStartTransaction(id string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.StartTransaction(request)
}

func (cs *CS) OnStopTransaction(id string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.StopTransaction(request)
}

func (cs *CS) OnDiagnosticsStatusNotification(id string, request *firmware.DiagnosticsStatusNotificationRequest) (confirmation *firmware.DiagnosticsStatusNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.DiagnosticStatusNotification(request)
}

func (cs *CS) OnFirmwareStatusNotification(id string, request *firmware.FirmwareStatusNotificationRequest) (confirmation *firmware.FirmwareStatusNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.FirmwareStatusNotification(request)
}
