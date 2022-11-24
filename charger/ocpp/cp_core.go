package ocpp

import (
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const (
	// messageExpiry     = 30 * time.Second
	transactionExpiry = time.Hour
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

// timestampValid returns false if status timestamps are outdated
func (cp *CP) isStatusTimestampValid(t time.Time) bool {
	// reject if expired
	if time.Since(t) > cp.timeout {
		return false
	}

	// assume having a timestamp is better than not
	if cp.status.Timestamp == nil {
		return true
	}

	// reject older values than we already have
	return !t.Before(cp.status.Timestamp.Time)
}

func (cp *CP) update() {
	cp.mu.Lock()
	cp.updated = time.Now()
	cp.mu.Unlock()
}

func (cp *CP) BootNotification(request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := &core.BootNotificationConfirmation{
		CurrentTime: types.NewDateTime(time.Now()),
		Interval:    60, // TODO
		Status:      core.RegistrationStatusAccepted,
	}

	// notify about boot event
	close(cp.bootC)

	return res, nil
}

func (cp *CP) StatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	if request == nil { // is this really neccessary?
		return nil, nil
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.updated = time.Now()

 	if request.Timestamp == nil || cp.isStatusTimestampValid(request.Timestamp.Time) {
		if cp.status == nil {
			close(cp.statusC) // signal initial status received
		}

		cp.status = request
		cp.statusUpdated = time.Now()
	} else {
		cp.log.TRACE.Printf("ignoring status: %s < %s", request.Timestamp.Time, cp.status.Timestamp)
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (cp *CP) Heartbeat(request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.updated = time.Now()

	res := &core.HeartbeatConfirmation{
		CurrentTime: types.NewDateTime(time.Now()),
	}

	return res, nil
}

func (cp *CP) StartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	if request == nil { // what!?
		return nil, nil
	}

	// we have to respond to every CP request
	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
		TransactionId: 1, // default
	}

	// properly filling in transaction Id
	if (cp.currentTransaction == nil) {
		res.TransactionId = cp.lastTransactionId+1
		cp.lastTransactionId = res.TransactionId
	} else	{
		res.TransactionId = cp.currentTransaction.Id
		// no need to save lastTransactionId because NewTransaction always does that
	}


	if (time.Since(request.Timestamp.Time) < transactionExpiry) || // too old transaction request
	(cp.currentTransaction != nil && cp.currentTransaction.Status() != TransactionStarting) { // we didnt start this transaction, invalidate it
		cp.log.DEBUG.Printf("start transaction: defering transaction stop request and hope for the best")
		defer Instance().RemoteStopTransaction(cp.ID(), func(resp *core.RemoteStopTransactionConfirmation, err error) {
				cp.log.TRACE.Printf("%T: %+v", resp, resp)
			}, res.TransactionId)
	}

	return res, nil
}

func (cp *CP) StopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// signal that transaction has ended
	if cp.currentTransaction != nil {
		if status := cp.currentTransaction.Status(); status != TransactionRunning || status != TransactionSuspended {
			cp.log.ERROR.Printf("stop transaction: stopping not started transaction")
		}

		// log mismatching id because we close transaction anyway
		if request.TransactionId != cp.currentTransaction.Id {
			cp.log.ERROR.Printf("stop transaction: mismatched id %d expected %d", request.TransactionId, cp.currentTransaction.Id)
		}

		cp.currentTransaction.SetStatus(TransactionFinished)
	} else {
		cp.log.WARN.Printf("stop transaction: stopping non existent transaction")
	}

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
	}

	return res, nil
}

// unsupported functions

func (cp *CP) DataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	res := &core.DataTransferConfirmation{
		Status: core.DataTransferStatusRejected,
	}

	return res, nil
}

func (cp *CP) DiagnosticStatusNotification(request *firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	return &firmware.DiagnosticsStatusNotificationConfirmation{}, nil
}

func (cp *CP) FirmwareStatusNotification(request *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)

	return &firmware.FirmwareStatusNotificationConfirmation{}, nil
}
