package ocpp

import (
	"errors"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidConnector   = errors.New("invalid connector")
	ErrInvalidTransaction = errors.New("invalid transaction")
)

func (cp *CP) OnBootNotification(request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	res := &core.BootNotificationConfirmation{
		CurrentTime: types.Now(),
		Interval:    60,
		Status:      core.RegistrationStatusAccepted,
	}

	cp.onceBoot.Do(func() {
		cp.bootNotificationRequestC <- request
	})

	return res, nil
}

func (cp *CP) OnStatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	if conn := cp.connectorByID(request.ConnectorId); conn != nil {
		return conn.OnStatusNotification(request)
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (cp *CP) OnMeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	// signal received
	select {
	case cp.meterC <- struct{}{}:
	default:
	}

	if conn := cp.connectorByID(request.ConnectorId); conn != nil {
		conn.OnMeterValues(request)
	}

	return new(core.MeterValuesConfirmation), nil
}

func (cp *CP) OnStartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	if conn := cp.connectorByID(request.ConnectorId); conn != nil {
		return conn.OnStartTransaction(request)
	}

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cp *CP) OnStopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}

	if conn := cp.connectorByTransactionID(request.TransactionId); conn != nil {
		return conn.OnStopTransaction(request)
	}

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept old pending stop message during startup
		},
	}

	return res, nil
}
