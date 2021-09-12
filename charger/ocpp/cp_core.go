package ocpp

import (
	"errors"

	core "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

func (cp *CP) Authorize(request *core.AuthorizeRequest) (confirmation *core.AuthorizeConfirmation, err error) {
	return nil, errors.New("not implemented")
}

func (cp *CP) BootNotification(request *core.BootNotificationRequest) (confirmation *core.BootNotificationConfirmation, err error) {
	return nil, errors.New("not implemented")
}

func (cp *CP) DataTransfer(request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	return nil, errors.New("not implemented")
}

func (cp *CP) Heartbeat(request *core.HeartbeatRequest) (confirmation *core.HeartbeatConfirmation, err error) {
	return nil, errors.New("not implemented")
}

func (cp *CP) MeterValues(request *core.MeterValuesRequest) (confirmation *core.MeterValuesConfirmation, err error) {
	return nil, errors.New("not implemented")
}

func (cp *CP) StatusNotification(request *core.StatusNotificationRequest) (confirmation *core.StatusNotificationConfirmation, err error) {
	return nil, errors.New("not implemented")
}

func (cp *CP) StartTransaction(request *core.StartTransactionRequest) (confirmation *core.StartTransactionConfirmation, err error) {
	return nil, errors.New("not implemented")
}

func (cp *CP) StopTransaction(request *core.StopTransactionRequest) (confirmation *core.StopTransactionConfirmation, err error) {
	return nil, errors.New("not implemented")
}
