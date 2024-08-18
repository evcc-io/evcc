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
	meterC  chan map[types.Measurand]types.SampledValue

	meterUpdated time.Time
	measurements map[types.Measurand]types.SampledValue
	timeout      time.Duration

	txnCount int // change initial value to the last known global transaction. Needs persistence
	txnId    int
	idTag    string
}

func NewConnector(log *util.Logger, id int, cp *CP, timeout time.Duration) (*Connector, error) {
	conn := &Connector{
		log:          log,
		cp:           cp,
		id:           id,
		clock:        clock.New(),
		statusC:      make(chan struct{}),
		measurements: make(map[types.Measurand]types.SampledValue),
		meterC:       make(chan map[types.Measurand]types.SampledValue),
		timeout:      timeout,
	}

	err := cp.registerConnector(id, conn)

	return conn, err
}

func (conn *Connector) TestClock(clock clock.Clock) {
	conn.clock = clock
}

func (conn *Connector) MeterSampled() <-chan map[types.Measurand]types.SampledValue {
	return conn.meterC
}

func (conn *Connector) ChargePoint() *CP {
	return conn.cp
}

func (conn *Connector) ID() int {
	return conn.id
}

func (conn *Connector) IdTag() string {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.idTag
}

func (conn *Connector) TriggerMessageRequest(feature remotetrigger.MessageTrigger, f ...func(request *remotetrigger.TriggerMessageRequest)) error {
	return Instance().TriggerMessageRequest(conn.cp.ID(), feature, func(request *remotetrigger.TriggerMessageRequest) {
		request.ConnectorId = &conn.id
		for _, f := range f {
			f(request)
		}
	})
}

// WatchDog triggers meter values messages if older than timeout.
// Must be wrapped in a goroutine.
func (conn *Connector) WatchDog(timeout time.Duration) {
	tick := time.NewTicker(2 * time.Second)
	for ; true; <-tick.C {
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
	timeout := time.After(conn.timeout)
	for {
		select {
		case <-conn.statusC:
			return nil

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

// Status returns the unmapped charge point status
func (conn *Connector) Status() (core.ChargePointStatus, error) {
	if !conn.cp.Connected() {
		return "", api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.status == nil {
		return core.ChargePointStatusUnavailable, nil
	}

	if conn.status.ErrorCode != core.NoError {
		return "", fmt.Errorf("%s: %s", conn.status.ErrorCode, conn.status.Info)
	}

	return conn.status.Status, nil
}

// NeedsAuthentication checks if local authentication or an initial RemoteStartTransaction is required
func (conn *Connector) NeedsAuthentication() bool {
	if !conn.cp.Connected() {
		return false
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	return conn.status != nil && conn.txnId == 0 && conn.status.Status == core.ChargePointStatusPreparing
}

// isMeterTimeout checks if meter values are outdated.
// Must only be called while holding lock.
func (conn *Connector) isMeterTimeout() bool {
	return conn.timeout > 0 && conn.clock.Since(conn.meterUpdated) > conn.timeout
}

var _ api.CurrentGetter = (*Connector)(nil)

// GetMaxCurrent returns the maximum phase current the charge point is set to offer
func (conn *Connector) GetMaxCurrent() (float64, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// fallthrough for last value on timeout when no transaction is running
	if conn.txnId != 0 && conn.isMeterTimeout() {
		return 0, api.ErrTimeout
	}

	if m, ok := conn.measurements[types.MeasurandCurrentOffered]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit) / 1e3, err
	}

	return 0, api.ErrNotAvailable
}

// GetMaxPower returns the maximum power the charge point is set to offer
func (conn *Connector) GetMaxPower() (float64, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// fallthrough for last value on timeout when no transaction is running
	if conn.txnId != 0 && conn.isMeterTimeout() {
		return 0, api.ErrTimeout
	}

	if m, ok := conn.measurements[types.MeasurandPowerOffered]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit), err
	}

	return 0, api.ErrNotAvailable
}

var _ api.Meter = (*Connector)(nil)

func (conn *Connector) CurrentPower() (float64, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// zero value on timeout when no transaction is running
	if conn.isMeterTimeout() {
		if conn.txnId != 0 {
			return 0, api.ErrTimeout
		}

		return 0, nil
	}

	if m, ok := conn.measurements[types.MeasurandPowerActiveImport]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit), err
	}

	return 0, api.ErrNotAvailable
}

func (conn *Connector) TotalEnergy() (float64, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// fallthrough for last value on timeout when no transaction is running
	if conn.txnId != 0 && conn.isMeterTimeout() {
		return 0, api.ErrTimeout
	}

	if m, ok := conn.measurements[types.MeasurandEnergyActiveImportRegister]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit) / 1e3, err
	}

	return 0, api.ErrNotAvailable
}

func (conn *Connector) Soc() (float64, error) {
	if !conn.cp.Connected() {
		return 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// fallthrough for last value on timeout when no transaction is running
	if conn.txnId != 0 && conn.isMeterTimeout() {
		return 0, api.ErrTimeout
	}

	if m, ok := conn.measurements[types.MeasurandSoC]; ok {
		return strconv.ParseFloat(m.Value, 64)
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

func getPhaseKey(key types.Measurand, phase int) types.Measurand {
	return key + types.Measurand(".L"+strconv.Itoa(phase))
}

func (conn *Connector) Currents() (float64, float64, float64, error) {
	if !conn.cp.Connected() {
		return 0, 0, 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// zero value on timeout when no transaction is running
	if conn.isMeterTimeout() {
		if conn.txnId != 0 {
			return 0, 0, 0, api.ErrTimeout
		}

		return 0, 0, 0, nil
	}

	currents := make([]float64, 0, 3)

	for phase := 1; phase <= 3; phase++ {
		m, ok := conn.measurements[getPhaseKey(types.MeasurandCurrentImport, phase)]
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

func (conn *Connector) Voltages() (float64, float64, float64, error) {
	if !conn.cp.Connected() {
		return 0, 0, 0, api.ErrTimeout
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// fallthrough for last value on timeout when no transaction is running
	if conn.txnId != 0 && conn.isMeterTimeout() {
		return 0, 0, 0, api.ErrTimeout
	}

	voltages := make([]float64, 0, 3)

	for phase := 1; phase <= 3; phase++ {
		m, ok := conn.measurements[getPhaseKey(types.MeasurandVoltage, phase)+"-N"]
		if !ok {
			// fallback for wrong voltage phase labeling
			m, ok = conn.measurements[getPhaseKey(types.MeasurandVoltage, phase)]
			if !ok {
				return 0, 0, 0, api.ErrNotAvailable
			}
		}

		f, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid voltage for phase %d: %w", phase, err)
		}

		voltages = append(voltages, scale(f, m.Unit))
	}

	return voltages[0], voltages[1], voltages[2], nil
}
