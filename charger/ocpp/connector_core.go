package ocpp

import (
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// timestampValid returns false if status timestamps are outdated
func (conn *Connector) timestampValid(t time.Time) bool {
	// reject if expired
	if time.Since(t) > messageExpiry {
		return false
	}

	// assume having a timestamp is better than not
	if conn.status.Timestamp == nil {
		return true
	}

	// reject older values than we already have
	return !t.Before(conn.status.Timestamp.Time)
}

func (conn *Connector) StatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.status == nil {
		conn.status = request
		close(conn.statusC) // signal initial status received
	} else if request.Timestamp == nil || conn.timestampValid(request.Timestamp.Time) {
		conn.status = request
	} else {
		conn.log.TRACE.Printf("ignoring status: %s < %s", request.Timestamp.Time, conn.status.Timestamp)
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (conn *Connector) MeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if request.TransactionId != nil && cp.txnId == 0 {
		conn.log.DEBUG.Printf("hijacking transaction: %d", *request.TransactionId)
		conn.txnId = *request.TransactionId
	}

	for _, meterValue := range request.MeterValue {
		// ignore old meter value requests
		if meterValue.Timestamp.Time.After(conn.meterUpdated) {
			for _, sample := range meterValue.SampledValue {
				conn.measurements[getSampleKey(sample)] = sample
				conn.meterUpdated = conn.clock.Now()
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

func (conn *Connector) StartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
		TransactionId: 1, // default
	}

	// create new transaction
	if request != nil && time.Since(request.Timestamp.Time) < transactionExpiry { // only respect transactions in the last hour
		conn.txnCount++
		res.TransactionId = conn.txnCount
	}

	conn.txnId = res.TransactionId

	return res, nil
}

func (conn *Connector) StopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// reset transaction
	if time.Since(request.Timestamp.Time) < transactionExpiry { // only respect transactions in the last hour
		// log mismatching id but close transaction anyway
		if request.TransactionId != conn.txnId {
			conn.log.ERROR.Printf("stop transaction: invalid id %d", request.TransactionId)
		}

		conn.txnId = 0
	}

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
	}

	return res, nil
}
