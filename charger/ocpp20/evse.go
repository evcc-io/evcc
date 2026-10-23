package ocpp20

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// Transaction represents an active OCPP 2.0.1 transaction
type Transaction struct {
	id            string
	chargingState transactions.ChargingState
	startTime     time.Time
	seqNo         int
}

// measurementKey identifies a unique measurement by measurand and phase
type measurementKey struct {
	measurand types.Measurand
	phase     types.Phase
}

// EVSE represents an OCPP 2.0.1 EVSE (Electric Vehicle Supply Equipment).
// connectorID selects the physical connector under this EVSE; 0 addresses the
// EVSE as a whole, positive values pick a specific connector for stations with
// multiple connectors per EVSE.
type EVSE struct {
	log         *util.Logger
	mu          sync.Mutex
	clock       clock.Clock
	cp          *Station
	id          int
	connectorID int // connector to target on this EVSE (0 = EVSE-wide)

	status     *availability.StatusNotificationRequest
	statusC    chan struct{}
	statusOnce sync.Once
	statusTime time.Time

	meterUpdated time.Time
	measurements map[measurementKey]types.SampledValue // latest sample per (measurand, phase)

	transaction   *Transaction
	idTag         string
	remoteIdTag   string
	meterInterval time.Duration
}

// NewEVSE creates a new EVSE for OCPP 2.0.1.
// connectorID selects the connector under this EVSE (0 = EVSE-wide).
func NewEVSE(ctx context.Context, log *util.Logger, id, connectorID int, cp *Station, idTag string, meterInterval time.Duration) (*EVSE, error) {
	evse := &EVSE{
		log:           log,
		clock:         clock.New(),
		cp:            cp,
		id:            id,
		connectorID:   connectorID,
		statusC:       make(chan struct{}),
		measurements:  make(map[measurementKey]types.SampledValue),
		meterInterval: meterInterval,
		remoteIdTag:   idTag,
	}

	if err := cp.registerEVSE(id, connectorID, evse); err != nil {
		return nil, err
	}

	// replay cached status if the station already sent one before the EVSE
	// was constructed (mirrors 1.6 connector.WithConnectorStatus behavior).
	Instance().WithEVSEStatus(cp.ID(), id, func(status *availability.StatusNotificationRequest) {
		evse.OnStatusNotification(status)
	})

	go func() {
		// deregister evse when the context is cancelled
		<-ctx.Done()
		cp.deregisterEVSE(id, connectorID)
	}()

	return evse, nil
}

// Initialized waits for initial status notification
func (e *EVSE) Initialized() error {
	// wait for initial status or timeout
	timeout := time.NewTimer(ocpp.Timeout)
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

// OnStatusNotification handles status notifications.
// Connector-level filtering is done by the CSMS dispatcher; this method
// unconditionally records what it is given.
func (e *EVSE) OnStatusNotification(request *availability.StatusNotificationRequest) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.status = request
	e.statusTime = time.Now()

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
		e.transaction = &Transaction{
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

// updateMeterValuesLocked updates meter values (must hold e.mu).
// Stores only the latest sample per (measurand, phase) to bound memory.
func (e *EVSE) updateMeterValuesLocked(meterValues []types.MeterValue) {
	for _, mv := range meterValues {
		for _, sv := range mv.SampledValue {
			e.measurements[measurementKey{sv.Measurand, sv.Phase}] = sv
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

// getMeasurement returns a measurement value with scaling applied.
// Tries the no-phase entry first, then falls back to any phase entry.
func (e *EVSE) getMeasurement(measurand types.Measurand) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if sv, ok := e.measurements[measurementKey{measurand: measurand}]; ok {
		return e.scale(sv), nil
	}
	for k, sv := range e.measurements {
		if k.measurand == measurand {
			return e.scale(sv), nil
		}
	}

	return 0, fmt.Errorf("measurement not available: %s", measurand)
}

// scale applies unit scaling for OCPP 2.0.1 SampledValue
func (e *EVSE) scale(sv types.SampledValue) float64 {
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
		if sv, ok := e.measurements[measurementKey{types.MeasurandCurrentImport, phase}]; ok {
			return e.scale(sv)
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
		if sv, ok := e.measurements[measurementKey{types.MeasurandVoltage, phase}]; ok {
			return e.scale(sv)
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
