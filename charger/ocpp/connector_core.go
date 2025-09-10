package ocpp

import (
	"slices"
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

	if conn.status == nil {
		conn.status = request
		close(conn.statusC) // signal initial status received
	} else if request.Timestamp == nil || conn.timestampValid(request.Timestamp.Time) {
		conn.status = request
	} else {
		conn.log.TRACE.Printf("ignoring status: %s < %s", request.Timestamp.Time, conn.status.Timestamp)
	}

	if conn.isWaitingForAuth() {
		if conn.remoteIdTag != "" {
			conn.RemoteStartTransactionRequest(conn.remoteIdTag)
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

func containsMeasurand(meterValues []types.MeterValue, measurand types.Measurand) bool {
	pos := slices.IndexFunc(meterValues, func(m types.MeterValue) bool {
		subpos := slices.IndexFunc(m.SampledValue, func(s types.SampledValue) bool {
			return s.Measurand == measurand
		})
		return subpos != -1
	})
	return pos != -1
}

func (conn *Connector) OnMeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if request.TransactionId != nil && *request.TransactionId > 0 &&
		conn.txnId == 0 && conn.status != nil &&
		(conn.status.Status == core.ChargePointStatusCharging ||
			conn.status.Status == core.ChargePointStatusSuspendedEV ||
			conn.status.Status == core.ChargePointStatusSuspendedEVSE) {
		conn.log.DEBUG.Printf("recovered transaction: %d", *request.TransactionId)
		conn.txnId = *request.TransactionId
	}

	// Some wallboxes do not report power if no car is connected
	// This leads to the situation that the last reported power value is used for current power consumption
	// which is wrong if the car has been disconnected in the meantime
	if !containsMeasurand(request.MeterValue, types.MeasurandPowerActiveImport) {
		conn.setPowerToZero(true)
	}

	for _, meterValue := range sortByAge(request.MeterValue) {
		if meterValue.Timestamp == nil {
			// this should be done before the sorting, but lets assume either all or no sample has a timestamp
			meterValue.Timestamp = types.NewDateTime(conn.clock.Now())
		}

		// ignore old meter value requests
		if !meterValue.Timestamp.Time.Before(conn.meterUpdated) {
			for _, sample := range meterValue.SampledValue {
				sample.Value = strings.TrimSpace(sample.Value)
				conn.measurements[getSampleKey(sample)] = sample
				conn.meterUpdated = meterValue.Timestamp.Time
			}
		}
	}

	return new(core.MeterValuesConfirmation), nil
}

func (conn *Connector) OnStartTransaction(request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.txnId = int(instance.txnId.Add(1))
	conn.idTag = request.IdTag

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
		TransactionId: conn.txnId,
	}

	return res, nil
}

func (conn *Connector) setPowerToZero(forceSetFallBackPower bool) {
	if _, ok := conn.measurements[types.MeasurandPowerActiveImport]; ok {
		conn.measurements[types.MeasurandPowerActiveImport] = types.SampledValue{
			Value: "0",
			Unit:  types.UnitOfMeasureW,
		}
	}

	for phase := 1; phase <= 3; phase++ {
		key := getPhaseKey(types.MeasurandPowerActiveImport, phase)
		if _, ok := conn.measurements[key]; ok {
			conn.measurements[key] = types.SampledValue{
				Value: "0",
				Unit:  types.UnitOfMeasureW,
			}
		}

		// Connector.CurrentPower() checks "phase-N" as matter of last resort
		// Can be always set to zero to have 0 power reported at startup, or CurrentPower() returns no Api error
		keyN := key + "-N"
		if _, ok := conn.measurements[keyN]; ok || forceSetFallBackPower {
			conn.measurements[keyN] = types.SampledValue{
				Value: "0",
				Unit:  types.UnitOfMeasureW,
			}
		}
	}
}

func (conn *Connector) assumeMeterStopped() {
	conn.meterUpdated = conn.clock.Now()
	conn.setPowerToZero(false)

	for phase := 1; phase <= 3; phase++ {
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
