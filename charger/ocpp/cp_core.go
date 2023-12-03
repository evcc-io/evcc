package ocpp

import (
	"errors"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const (
	messageExpiry     = 30 * time.Second
	transactionExpiry = time.Hour
)

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidConnector   = errors.New("invalid connector")
	ErrInvalidTransaction = errors.New("invalid transaction")
)

func (cp *CP) Authorize(request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	// TODO check if this authorizes foreign RFID tags
	res := &core.AuthorizeConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cp *CP) BootNotification(request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	res := &core.BootNotificationConfirmation{
		CurrentTime: types.Now(),
		Interval:    60, // TODO
		Status:      core.RegistrationStatusAccepted,
	}

	return res, nil
}

func (cp *CP) DiagnosticStatusNotification(request *firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
	return new(firmware.DiagnosticsStatusNotificationConfirmation), nil
}

func (cp *CP) FirmwareStatusNotification(request *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error) {
	return new(firmware.FirmwareStatusNotificationConfirmation), nil
}

func (cp *CP) StatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	conn := cp.connectorByID(request.ConnectorId)
	if conn == nil {
		return nil, ErrInvalidConnector
	}

	return conn.StatusNotification(request)
}

func (cp *CP) DataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	res := &core.DataTransferConfirmation{
		Status: core.DataTransferStatusAccepted,
	}

	return res, nil
}

func (cp *CP) Heartbeat(request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	res := &core.HeartbeatConfirmation{
		CurrentTime: types.Now(),
	}

	return res, nil
}

func (cp *CP) MeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	conn := cp.connectorByID(request.ConnectorId)
	if conn == nil {
		return nil, ErrInvalidConnector
	}

	return conn.MeterValues(request)
}

func (cp *CP) StartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	conn := cp.connectorByID(request.ConnectorId)
	if conn == nil {
		return nil, ErrInvalidConnector
	}

	return conn.StartTransaction(request)
}

func (cp *CP) StopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	conn := cp.connectorByTransactionID(request.TransactionId)
	if conn == nil {
		return nil, ErrInvalidTransaction
	}

	return conn.StopTransaction(request)
}
