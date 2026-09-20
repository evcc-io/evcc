package ocpp

import (
	"strings"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// timestampValid returns false if status timestamps are outdated
func (conn *Connector) timestampValid(t time.Time) bool {
	// reject if expired
	if conn.clock.Since(t) > Timeout {
		return false
	}

	// assume having a timestamp is better than not
	if conn.status.Timestamp == nil {
		return true
	}

	// reject older values than we already have
	return !t.Before(conn.status.Timestamp.Time)
}

func (conn *Connector) OnStatusNotification(request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	var applied bool
	if conn.status == nil {
		conn.status = request
		close(conn.statusC) // signal initial status received
		applied = true
	} else if request.Timestamp == nil || conn.timestampValid(request.Timestamp.Time) {
		conn.status = request
		applied = true
	} else {
		conn.log.TRACE.Printf("ignoring status: %s < %s", request.Timestamp.Time, conn.status.Timestamp)
	}

	// a freshly applied status supersedes anything cached from before a reboot
	if applied {
		conn.statusStale = false
	}

	// Available means cable unplugged and any prior transaction is stale
	if applied && request.Status == core.ChargePointStatusAvailable {
		conn.clearTransaction("Available status")
	}

	if conn.isWaitingForAuth() {
		if conn.remoteIdTag != "" {
			// dispatch asynchronously: RemoteStartTransactionRequest issues a
			// synchronous CS→CP request whose response is read by this same
			// goroutine, so a blocking call would deadlock the WebSocket read
			// loop (cf. ocpp_test_handler.go).
			go func(idTag string) {
				if err := conn.RemoteStartTransactionRequest(idTag); err != nil {
					conn.log.ERROR.Printf("RemoteStartTransaction: %v", err)
				}
			}(conn.remoteIdTag)
		} else {
			conn.log.DEBUG.Printf("waiting for local authentication")
		}
	}

	return new(core.StatusNotificationConfirmation), nil
}

func getSampleKey(s types.SampledValue) types.Measurand {
	if s.Phase != "" {
		return s.Measurand + types.Measurand("."+string(s.Phase))
	}

	return s.Measurand
}

// isBoundaryContext returns true for readings that are a snapshot taken at the
// transaction's start or end rather than a live measurement. Context is optional
// and defaults to Sample.Periodic.
func isBoundaryContext(c types.ReadingContext) bool {
	return c == types.ReadingContextTransactionBegin || c == types.ReadingContextTransactionEnd
}

func (conn *Connector) OnMeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if request.TransactionId != nil && *request.TransactionId > 0 &&
		conn.txnId == 0 && conn.status != nil && !conn.statusStale &&
		(conn.status.Status == core.ChargePointStatusCharging ||
			conn.status.Status == core.ChargePointStatusSuspendedEV ||
			conn.status.Status == core.ChargePointStatusSuspendedEVSE) {
		conn.log.DEBUG.Printf("recovered transaction: %d", *request.TransactionId)
		conn.txnId = *request.TransactionId
	}

	for _, meterValue := range sortByAge(request.MeterValue) {
		if meterValue.Timestamp == nil {
			// this should be done before the sorting, but lets assume either all or no sample has a timestamp
			meterValue.Timestamp = types.NewDateTime(conn.clock.Now())
		}

		// ignore old meter value requests
		if !meterValue.Timestamp.Time.Before(conn.meterUpdated) {
			// a charge point may repeat a measurand with a different context, e.g. a live
			// Sample.Periodic value next to a static Transaction.Begin snapshot that never
			// changes. Apply boundary snapshots first so live readings win independent of
			// their order within the message.
			for _, boundary := range []bool{true, false} {
				for _, sample := range meterValue.SampledValue {
					if isBoundaryContext(sample.Context) != boundary {
						continue
					}

					sample.Value = strings.TrimSpace(sample.Value)
					conn.measurements[getSampleKey(sample)] = sample
					conn.meterUpdated = meterValue.Timestamp.Time
				}
			}
		}
	}

	return new(core.MeterValuesConfirmation), nil
}

func (conn *Connector) OnStartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.txnId = int(conn.cp.cs.txnId.Add(1))
	conn.idTag = request.IdTag

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
		TransactionId: conn.txnId,
	}

	return res, nil
}

// clearTransaction resets transaction state and zeroes reported power.
// Must be called with conn.mu held. No-op if no transaction is tracked.
func (conn *Connector) clearTransaction(reason string) {
	if conn.txnId == 0 {
		return
	}

	conn.log.DEBUG.Printf("clearing stale transaction %d on %s", conn.txnId, reason)
	conn.txnId = 0
	conn.idTag = ""
	conn.assumeMeterStopped()
}

// resetTransaction clears any transaction state after a charge point reboot.
// A reboot ends every transaction the central system still tracked; without
// this a stale txnId keeps isWaitingForAuth false and suppresses the automatic
// RemoteStartTransaction when the connector reconnects straight into Preparing
// (i.e. never reports Available, e.g. Grizzl-E). It also marks the cached
// status stale so it cannot qualify a transaction for recovery.
func (conn *Connector) resetTransaction() {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// the status cached from before the reboot must not qualify a MeterValues
	// transaction id for recovery until the charge point reports a fresh one.
	// Set outside clearTransaction, which is a no-op when no transaction is
	// tracked - and txnId == 0 is exactly the recovery precondition.
	conn.statusStale = true

	conn.clearTransaction("reboot")
}

func (conn *Connector) assumeMeterStopped() {
	conn.meterUpdated = conn.clock.Now()

	if _, ok := conn.measurements[types.MeasurandPowerActiveImport]; ok {
		conn.measurements[types.MeasurandPowerActiveImport] = types.SampledValue{
			Value: "0",
			Unit:  types.UnitOfMeasureW,
		}
	}

	for phase := 1; phase <= 3; phase++ {
		// phase powers
		for _, suffix := range []types.Measurand{"", "-N"} {
			key := getPhaseKey(types.MeasurandPowerActiveImport, phase) + suffix
			if _, ok := conn.measurements[key]; ok {
				conn.measurements[key] = types.SampledValue{
					Value: "0",
					Unit:  types.UnitOfMeasureW,
				}
			}
		}

		// phase currents
		key := getPhaseKey(types.MeasurandCurrentImport, phase)
		if _, ok := conn.measurements[key]; ok {
			conn.measurements[key] = types.SampledValue{
				Value: "0",
				Unit:  types.UnitOfMeasureA,
			}
		}
	}
}

func (conn *Connector) OnStopTransaction(request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.txnId = 0
	conn.idTag = ""

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept
		},
	}

	conn.assumeMeterStopped()

	return res, nil
}
