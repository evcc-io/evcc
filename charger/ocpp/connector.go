package ocpp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type Connector struct {
	log   *util.Logger
	mu    sync.Mutex
	clock clock.Clock // mockable time
	cp    *CP
	id    int

	status  *core.StatusNotificationRequest
	statusC chan struct{}

	meterUpdated time.Time
	measurements map[string]types.SampledValue
	timeout      time.Duration

	txnCount int // change initial value to the last known global transaction. Needs persistence
	txnId    int
}

func NewConnector(log *util.Logger, id int, cp *CP, timeout time.Duration) (*Connector, error) {
	conn := &Connector{
		log:          log,
		cp:           cp,
		id:           id,
		clock:        clock.New(),
		statusC:      make(chan struct{}),
		measurements: make(map[string]types.SampledValue),
		timeout:      timeout,
	}

	err := cp.registerConnector(id, conn)

	return conn, err
}

func (conn *Connector) TestClock(clock clock.Clock) {
	conn.clock = clock
}

func (conn *Connector) ChargePoint() *CP {
	return conn.cp
}

func (conn *Connector) ID() int {
	return conn.id
}

func (conn *Connector) TriggerMessageRequest(feature remotetrigger.MessageTrigger, f ...func(request *remotetrigger.TriggerMessageRequest)) {
	Instance().TriggerMessageRequest(conn.cp.ID(), feature, func(request *remotetrigger.TriggerMessageRequest) {
		request.ConnectorId = &conn.id
		for _, f := range f {
			f(request)
		}
	})
}

// WatchDog triggers meter values messages if older than timeout.
// Must be wrapped in a goroutine.
func (conn *Connector) WatchDog(timeout time.Duration) {
	for ; true; <-time.Tick(timeout) {
		conn.mu.Lock()
		update := conn.txnId != 0 && conn.clock.Since(conn.meterUpdated) > timeout
		conn.mu.Unlock()

		if update {
			conn.TriggerMessageRequest(core.MeterValuesFeatureName)
		}
	}
}

// Initialized waits for initial charge point status notification
func (conn *Connector) Initialized() error {
	trigger := time.After(conn.timeout / 2)
	timeout := time.After(conn.timeout)
	for {
		select {
		case <-conn.statusC:
			return nil

		case <-trigger:
			conn.TriggerMessageRequest(core.StatusNotificationFeatureName)

		case <-timeout:
			return api.ErrTimeout
		}
	}
}

// TransactionID returns the current transaction id
func (conn *Connector) TransactionID() (int, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	return conn.txnId, nil
}

func (conn *Connector) Status() (api.ChargeStatus, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	res := api.StatusNone

	if !conn.cp.Connected() {
		return res, api.ErrTimeout
	}

	if conn.status.ErrorCode != core.NoError {
		return res, fmt.Errorf("%s: %s", conn.status.ErrorCode, conn.status.Info)
	}

	switch conn.status.Status {
	case core.ChargePointStatusAvailable, // "Available"
		core.ChargePointStatusUnavailable: // "Unavailable"
		res = api.StatusA
	case
		core.ChargePointStatusPreparing,     // "Preparing"
		core.ChargePointStatusSuspendedEVSE, // "SuspendedEVSE"
		core.ChargePointStatusSuspendedEV,   // "SuspendedEV"
		core.ChargePointStatusFinishing:     // "Finishing"
		res = api.StatusB
	case core.ChargePointStatusCharging: // "Charging"
		res = api.StatusC
	case core.ChargePointStatusReserved, // "Reserved"
		core.ChargePointStatusFaulted: // "Faulted"
		return api.StatusF, fmt.Errorf("chargepoint status: %s", conn.status.ErrorCode)
	default:
		return api.StatusNone, fmt.Errorf("invalid chargepoint status: %s", conn.status.Status)
	}

	return res, nil
}

// isMeterTimeout checks if meter values are outdated.
// Must only be called while holding lock.
func (conn *Connector) isMeterTimeout() bool {
	return conn.timeout > 0 && conn.clock.Since(conn.meterUpdated) > conn.timeout
}

var _ api.Meter = (*Connector)(nil)

func (conn *Connector) CurrentPower() (float64, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// zero value on timeout when not charging
	if conn.isMeterTimeout() {
		if conn.txnId != 0 {
			return 0, api.ErrTimeout
		}

		return 0, nil
	}

	if m, ok := conn.measurements[string(types.MeasurandPowerActiveImport)]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit), err
	}

	return 0, api.ErrNotAvailable
}

var _ api.MeterEnergy = (*Connector)(nil)

func (conn *Connector) TotalEnergy() (float64, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// fallthrough for last value on timeout when not charging
	if conn.txnId != 0 && conn.isMeterTimeout() {
		return 0, api.ErrTimeout
	}

	if m, ok := conn.measurements[string(types.MeasurandEnergyActiveImportRegister)]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit) / 1e3, err
	}

	return 0, api.ErrNotAvailable
}

func scale(f float64, scale types.UnitOfMeasure) float64 {
	switch {
	case strings.HasPrefix(string(scale), "k"):
		return f * 1e3
	case strings.HasPrefix(string(scale), "m"):
		return f / 1e3
	default:
		return f
	}
}

func getKeyCurrentPhase(phase int) string {
	return string(types.MeasurandCurrentImport) + "@L" + strconv.Itoa(phase)
}

var _ api.PhaseCurrents = (*Connector)(nil)

func (conn *Connector) Currents() (float64, float64, float64, error) {
	if !conn.cp.Connected() {
		return 0, 0, 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// zero value on timeout when not charging
	if conn.isMeterTimeout() {
		if conn.txnId != 0 {
			return 0, 0, 0, api.ErrTimeout
		}

		return 0, 0, 0, nil
	}

	currents := make([]float64, 0, 3)

	for phase := 1; phase <= 3; phase++ {
		m, ok := conn.measurements[getKeyCurrentPhase(phase)]
		if !ok {
			return 0, 0, 0, api.ErrNotAvailable
		}

		f, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid current for phase %d: %w", phase, err)
		}

		currents = append(currents, scale(f, m.Unit))
	}

	return currents[0], currents[1], currents[2], nil
}
