package ocpp

import (
	"sync/atomic"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func (cp *CP) Authorize(request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	// TODO check if this authorizes foreign RFID tags

	res := &core.AuthorizeConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cp *CP) BootNotification(request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	if request != nil {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		cp.boot = request
		cp.updated.Broadcast()
	}

	res := &core.BootNotificationConfirmation{
		CurrentTime: types.NewDateTime(time.Now()),
		Interval:    60, // TODO
		Status:      core.RegistrationStatusAccepted,
	}

	return res, nil
}

func (cp *CP) StatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	if request != nil {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		cp.status = request
		cp.updated.Broadcast()
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (cp *CP) DataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	res := &core.DataTransferConfirmation{
		Status: core.DataTransferStatusRejected,
	}

	return res, nil
}

func (cp *CP) Heartbeat(request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	res := &core.HeartbeatConfirmation{
		CurrentTime: types.NewDateTime(time.Now()),
	}

	return res, nil
}

func (cp *CP) MeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	if request != nil {
		cp.mu.Lock()
		cp.meterValues = request.MeterValue
		cp.mu.Unlock()
	}

	return new(core.MeterValuesConfirmation), nil
}

func (cp *CP) StartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	// create new transaction
	txn := atomic.AddInt64(&cp.txnCount, 1)

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
		TransactionId: int(txn),
	}

	return res, nil
}

func (cp *CP) StopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
	}

	return res, nil
}
