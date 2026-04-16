package ocpp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// Transaction20 represents an active OCPP 2.0.1 transaction
type Transaction20 struct {
	id            string
	chargingState transactions.ChargingState
	startTime     time.Time
	seqNo         int
}

// EVSE represents an OCPP 2.0.1 EVSE (Electric Vehicle Supply Equipment)
type EVSE struct {
	log   *util.Logger
	mu    sync.Mutex
	clock clock.Clock
	cp    *CS20CP
	id    int

	status      *availability.StatusNotificationRequest
	statusC     chan struct{}
	statusOnce  sync.Once
	statusTime  time.Time
	connectorID int // connector ID from last status notification

	meterUpdated time.Time
	measurements map[types.Measurand][]types.SampledValue // stores all samples including per-phase

	transaction   *Transaction20
	idTag         string
	remoteIdTag   string
	meterInterval time.Duration
}

// NewEVSE creates a new EVSE for OCPP 2.0.1
func NewEVSE(ctx context.Context, log *util.Logger, id int, cp *CS20CP, idTag string, meterInterval time.Duration) (*EVSE, error) {
	evse := &EVSE{
		log:           log,
		clock:         clock.New(),
		cp:            cp,
		id:            id,
		statusC:       make(chan struct{}),
		measurements:  make(map[types.Measurand][]types.SampledValue),
		meterInterval: meterInterval,
		remoteIdTag:   idTag,
	}

	if err := cp.registerEVSE(id, evse); err != nil {
		return nil, err
	}

	return evse, nil
}

// Initialized waits for initial status notification
func (e *EVSE) Initialized() error {
	// wait for initial status or timeout
	timeout := time.NewTimer(Timeout)
	defer timeout.Stop()

	select {
	case <-e.statusC:
		return nil
	case <-timeout.C:
		return fmt.Errorf("evse %d: timeout waiting for initial status", e.id)
	}
}

// WatchDog triggers meter values periodically
func (e *EVSE) WatchDog(ctx context.Context, meterInterval time.Duration) {
	// TODO: Implement watchdog for OCPP 2.0.1
	// In 2.0.1, meter values typically come via TransactionEvent
}

// OnStatusNotification handles status notifications
func (e *EVSE) OnStatusNotification(request *availability.StatusNotificationRequest) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.status = request
	e.statusTime = time.Now()
	e.connectorID = request.ConnectorID

	// signal initialization (once only, safe for concurrent calls)
	e.statusOnce.Do(func() {
		close(e.statusC)
	})
}

// OnTransactionEvent handles transaction events
func (e *EVSE) OnTransactionEvent(request *transactions.TransactionEventRequest) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch request.EventType {
	case transactions.TransactionEventStarted:
		e.transaction = &Transaction20{
			id:            request.TransactionInfo.TransactionID,
			chargingState: request.TransactionInfo.ChargingState,
			startTime:     request.Timestamp.Time,
			seqNo:         request.SequenceNo,
		}
		if request.IDToken != nil {
			e.idTag = request.IDToken.IdToken
		}
		e.log.DEBUG.Printf("evse %d: transaction started: %s", e.id, e.transaction.id)

	case transactions.TransactionEventUpdated:
		if e.transaction != nil {
			e.transaction.seqNo = request.SequenceNo
			e.transaction.chargingState = request.TransactionInfo.ChargingState
		}

	case transactions.TransactionEventEnded:
		e.log.DEBUG.Printf("evse %d: transaction ended: %s", e.id, request.TransactionInfo.TransactionID)
		e.transaction = nil
		e.idTag = ""
	}

	// update meter values from transaction event
	if len(request.MeterValue) > 0 {
		e.updateMeterValuesLocked(request.MeterValue)
	}
}

// updateMeterValues updates meter values (thread-safe wrapper)
func (e *EVSE) updateMeterValues(meterValues []types.MeterValue) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.updateMeterValuesLocked(meterValues)
}

// updateMeterValuesLocked updates meter values (must hold e.mu)
func (e *EVSE) updateMeterValuesLocked(meterValues []types.MeterValue) {
	for _, mv := range meterValues {
		for _, sv := range mv.SampledValue {
			e.measurements[sv.Measurand] = append(e.measurements[sv.Measurand], sv)
		}
	}
	e.meterUpdated = time.Now()
}

// Status returns the current connector status
func (e *EVSE) Status() (availability.ConnectorStatus, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status == nil {
		return availability.ConnectorStatusUnavailable, errors.New("no status")
	}

	return e.status.ConnectorStatus, nil
}

// ChargingState returns the current charging state from the transaction
func (e *EVSE) ChargingState() (transactions.ChargingState, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.transaction == nil {
		return "", false
	}

	return e.transaction.chargingState, true
}

// TransactionID returns the current transaction ID
func (e *EVSE) TransactionID() (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.transaction == nil {
		return "", errors.New("no transaction")
	}

	return e.transaction.id, nil
}

// IdTag returns the current IdTag
func (e *EVSE) IdTag() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.idTag != "" {
		return e.idTag
	}
	return e.remoteIdTag
}

// NeedsAuthentication returns true if the EVSE requires authentication
func (e *EVSE) NeedsAuthentication() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if status is Occupied without a transaction
	if e.status != nil && e.status.ConnectorStatus == availability.ConnectorStatusOccupied {
		return e.transaction == nil
	}

	return false
}

// getMeasurement returns a measurement value with scaling applied (uses latest sample)
func (e *EVSE) getMeasurement(measurand types.Measurand) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if samples, ok := e.measurements[measurand]; ok && len(samples) > 0 {
		// return the latest sample (last in slice)
		return e.scale20(samples[len(samples)-1]), nil
	}

	return 0, fmt.Errorf("measurement not available: %s", measurand)
}

// scale20 applies unit scaling for OCPP 2.0.1 SampledValue
func (e *EVSE) scale20(sv types.SampledValue) float64 {
	val := sv.Value

	if sv.UnitOfMeasure != nil && sv.UnitOfMeasure.Multiplier != nil {
		val *= math.Pow(10, float64(*sv.UnitOfMeasure.Multiplier))
	}

	return val
}

// CurrentPower returns the current power in W
func (e *EVSE) CurrentPower() (float64, error) {
	return e.getMeasurement(types.MeasurandPowerActiveImport)
}

// TotalEnergy returns the total energy in kWh
func (e *EVSE) TotalEnergy() (float64, error) {
	val, err := e.getMeasurement(types.MeasurandEnergyActiveImportRegister)
	return val / 1000, err // OCPP reports Wh, evcc uses kWh
}

// Soc returns the vehicle SoC
func (e *EVSE) Soc() (float64, error) {
	return e.getMeasurement(types.MeasurandSoC)
}

// Currents returns the phase currents
func (e *EVSE) Currents() (float64, float64, float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	getPhase := func(phase types.Phase) float64 {
		if samples, ok := e.measurements[types.MeasurandCurrentImport]; ok {
			for _, sv := range samples {
				if sv.Phase == phase {
					return e.scale20(sv)
				}
			}
		}
		return 0
	}

	return getPhase(types.PhaseL1), getPhase(types.PhaseL2), getPhase(types.PhaseL3), nil
}

// Voltages returns the phase voltages
func (e *EVSE) Voltages() (float64, float64, float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	getPhase := func(phase types.Phase) float64 {
		if samples, ok := e.measurements[types.MeasurandVoltage]; ok {
			for _, sv := range samples {
				if sv.Phase == phase {
					return e.scale20(sv)
				}
			}
		}
		return 0
	}

	return getPhase(types.PhaseL1), getPhase(types.PhaseL2), getPhase(types.PhaseL3), nil
}

// GetMaxCurrent returns the offered current
func (e *EVSE) GetMaxCurrent() (float64, error) {
	return e.getMeasurement(types.MeasurandCurrentOffered)
}

// GetMaxPower returns the offered power
func (e *EVSE) GetMaxPower() (float64, error) {
	return e.getMeasurement(types.MeasurandPowerOffered)
}

// ID returns the EVSE ID
func (e *EVSE) ID() int {
	return e.id
}
