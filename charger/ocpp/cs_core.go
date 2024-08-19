package ocpp

import (
	"errors"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
)

// cs actions

func (cs *CS) TriggerResetRequest(id string, resetType core.ResetType) error {
	rc := make(chan error, 1)

	err := cs.Reset(id, func(request *core.ResetConfirmation, err error) {
		if err == nil && request != nil && request.Status != core.ResetStatusAccepted {
			err = errors.New(string(request.Status))
		}

		rc <- err
	}, resetType)

	return Wait(err, rc, cs.timeout)
}

func (cs *CS) TriggerMessageRequest(id string, requestedMessage remotetrigger.MessageTrigger, props ...func(request *remotetrigger.TriggerMessageRequest)) error {
	rc := make(chan error, 1)

	err := cs.TriggerMessage(id, func(request *remotetrigger.TriggerMessageConfirmation, err error) {
		if err == nil && request != nil && request.Status != remotetrigger.TriggerMessageStatusAccepted {
			err = errors.New(string(request.Status))
		}

		rc <- err
	}, requestedMessage, props...)

	return Wait(err, rc, cs.timeout)
}

// cp actions

func (cs *CS) OnAuthorize(id string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.Authorize(request)
}

func (cs *CS) OnBootNotification(id string, request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.BootNotification(request)
}

func (cs *CS) OnDataTransfer(id string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.DataTransfer(request)
}

func (cs *CS) OnHeartbeat(id string, request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.Heartbeat(request)
}

func (cs *CS) OnMeterValues(id string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.MeterValues(request)
}

func (cs *CS) OnStatusNotification(id string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.StatusNotification(request)
}

func (cs *CS) OnStartTransaction(id string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.StartTransaction(request)
}

func (cs *CS) OnStopTransaction(id string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.StopTransaction(request)
}

func (cs *CS) OnDiagnosticsStatusNotification(id string, request *firmware.DiagnosticsStatusNotificationRequest) (confirmation *firmware.DiagnosticsStatusNotificationConfirmation, err error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.DiagnosticStatusNotification(request)
}

func (cs *CS) OnFirmwareStatusNotification(id string, request *firmware.FirmwareStatusNotificationRequest) (confirmation *firmware.FirmwareStatusNotificationConfirmation, err error) {
	cp, err := cs.ChargepointByID(id)
	if err != nil {
		return nil, err
	}

	return cp.FirmwareStatusNotification(request)
}
