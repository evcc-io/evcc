package ocpp

import (
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const (
	messageExpiry     = 30 * time.Second
	transactionExpiry = time.Hour
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
		CurrentTime: types.NewDateTime(cp.clock.Now()),
		Interval:    60, // TODO
		Status:      core.RegistrationStatusAccepted,
	}

	return res, nil
}

// timestampValid returns false if status timestamps are outdated
func (cp *CP) timestampValid(t time.Time) bool {
	// reject if expired
	if time.Since(t) > messageExpiry {
		return false
	}

	// assume having a timestamp is better than not
	if cp.status.Timestamp == nil {
		return true
	}

	// reject older values than we already have
	return !t.Before(cp.status.Timestamp.Time)
}

func (cp *CP) StatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	if request != nil && request.ConnectorId == cp.connector {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		if cp.status == nil {
			cp.status = request
			close(cp.statusC) // signal initial status received
		} else if request.Timestamp == nil || cp.timestampValid(request.Timestamp.Time) {
			cp.status = request
		} else {
			cp.log.TRACE.Printf("ignoring status: %s < %s", request.Timestamp.Time, cp.status.Timestamp)
		}
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (cp *CP) DataTransfer(request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	res := &core.DataTransferConfirmation{
		Status: core.DataTransferStatusAccepted,
	}

	return res, nil
}

func (cp *CP) Heartbeat(request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	res := &core.HeartbeatConfirmation{
		CurrentTime: types.NewDateTime(cp.clock.Now()),
	}

	return res, nil
}

func (cp *CP) MeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	if request != nil && request.ConnectorId == cp.connector {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		if request.TransactionId != nil && cp.txnId == 0 {
			cp.log.DEBUG.Printf("hijacking transaction: %d", *request.TransactionId)
			cp.txnId = *request.TransactionId
		}

		for _, meterValue := range request.MeterValue {
			// ignore old meter value requests
			if meterValue.Timestamp.Time.After(cp.meterUpdated) {
				for _, sample := range meterValue.SampledValue {
					cp.measurements[getSampleKey(sample)] = sample
					cp.meterUpdated = cp.clock.Now()
				}
			}
		}
	}

	return new(core.MeterValuesConfirmation), nil
}

func getSampleKey(s types.SampledValue) string {
	if s.Phase != "" {
		return string(s.Measurand) + "@" + string(s.Phase)
	}

	return string(s.Measurand)
}

func (cp *CP) StartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	if request == nil || request.ConnectorId != cp.connector {
		return new(core.StartTransactionConfirmation), nil
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
		TransactionId: 1, // default
	}

	// create new transaction
	if request != nil && time.Since(request.Timestamp.Time) < transactionExpiry { // only respect transactions in the last hour
		cp.txnCount++
		res.TransactionId = cp.txnCount
	}

	cp.txnId = res.TransactionId

	return res, nil
}

func (cp *CP) StopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	if request != nil {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		// reset transaction
		if time.Since(request.Timestamp.Time) < transactionExpiry { // only respect transactions in the last hour
			// log mismatching id but close transaction anyway
			if request.TransactionId != cp.txnId {
				cp.log.ERROR.Printf("stop transaction: invalid id %d", request.TransactionId)
			}

			cp.txnId = 0
		}
	}

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
	}

	return res, nil
}

func (cp *CP) DiagnosticStatusNotification(request *firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
	return &firmware.DiagnosticsStatusNotificationConfirmation{}, nil
}

func (cp *CP) FirmwareStatusNotification(request *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error) {
	return &firmware.FirmwareStatusNotificationConfirmation{}, nil
}
