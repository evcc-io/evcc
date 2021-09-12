package ocpp

import (
	core "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

func (cs *CS) OnAuthorize(chargePointId string, request *core.AuthorizeRequest) (confirmation *core.AuthorizeConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.Authorize(request)
}

func (cs *CS) OnBootNotification(chargePointId string, request *core.BootNotificationRequest) (confirmation *core.BootNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.BootNotification(request)
}

func (cs *CS) OnDataTransfer(chargePointId string, request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.DataTransfer(request)
}

func (cs *CS) OnHeartbeat(chargePointId string, request *core.HeartbeatRequest) (confirmation *core.HeartbeatConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.Heartbeat(request)
}

func (cs *CS) OnMeterValues(chargePointId string, request *core.MeterValuesRequest) (confirmation *core.MeterValuesConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.MeterValues(request)
}

func (cs *CS) OnStatusNotification(chargePointId string, request *core.StatusNotificationRequest) (confirmation *core.StatusNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StatusNotification(request)
}

func (cs *CS) OnStartTransaction(chargePointId string, request *core.StartTransactionRequest) (confirmation *core.StartTransactionConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StartTransaction(request)
}

func (cs *CS) OnStopTransaction(chargePointId string, request *core.StopTransactionRequest) (confirmation *core.StopTransactionConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StopTransaction(request)
}
