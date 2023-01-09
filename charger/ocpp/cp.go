package ocpp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const (
	// Core profile keys
	KeyNumberOfConnectors = "NumberOfConnectors"

	// Meter profile keys
	KeyMeterValuesSampledData   = "MeterValuesSampledData"
	KeyMeterValueSampleInterval = "MeterValueSampleInterval"

	// Smart Charging profile keys
	KeyChargeProfileMaxStackLevel              = "ChargeProfileMaxStackLevel"
	KeyChargingScheduleAllowedChargingRateUnit = "ChargingScheduleAllowedChargingRateUnit"
	KeyChargingScheduleMaxPeriods              = "ChargingScheduleMaxPeriods"
	KeyConnectorSwitch3to1PhaseSupported       = "ConnectorSwitch3to1PhaseSupported"
	KeyMaxChargingProfilesInstalled            = "MaxChargingProfilesInstalled"

	// Alfen specific keys
	KeyAlfenPlugAndChargeIdentifier = "PlugAndChargeIdentifier"
)

type CP struct {
	mu   sync.Mutex
	log  *util.Logger
	once sync.Once

	id string

	connectC, statusC chan struct{}
	updated           time.Time
	status            *core.StatusNotificationRequest

	timeout      time.Duration
	meterUpdated time.Time
	measurements map[string]types.SampledValue

	txnCount int // change initial value to the last known global transaction. Needs persistence
	txnId    int
}

func NewChargePoint(log *util.Logger, id string, timeout time.Duration) *CP {
	return &CP{
		log:          log,
		id:           id,
		connectC:     make(chan struct{}),
		statusC:      make(chan struct{}),
		measurements: make(map[string]types.SampledValue),
		timeout:      timeout,
	}
}

func (cp *CP) ID() string {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.id
}

func (cp *CP) RegisterID(id string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.id != "" {
		panic("ocpp: cannot re-register id")
	}

	cp.id = id
}

func (cp *CP) Connect() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.once.Do(func() {
		close(cp.connectC)
	})
}

func (cp *CP) HasConnected() <-chan struct{} {
	return cp.connectC
}

func (cp *CP) Initialized(timeout time.Duration) bool {
	cp.log.DEBUG.Printf("waiting for chargepoint status: %v", timeout)

	// trigger status
	time.AfterFunc(5*time.Second, func() {
		select {
		case <-cp.statusC:
			return
		default:
			Instance().TriggerMessageRequest(cp.ID(), core.StatusNotificationFeatureName)
		}
	})

	// wait for status
	select {
	case <-cp.statusC:
		cp.update()
		return true
	case <-time.After(timeout):
		return false
	}
}

// WatchDog triggers meter values messages if older than timeout.
// Must be wrapped in a goroutine.
func (cp *CP) WatchDog(timeout time.Duration) {
	for ; true; <-time.NewTicker(timeout).C {
		cp.mu.Lock()
		update := cp.txnId != 0 && time.Since(cp.meterUpdated) > timeout
		cp.mu.Unlock()

		if update {
			Instance().TriggerMessageRequest(cp.ID(), core.MeterValuesFeatureName)
		}
	}
}

// TransactionID returns the current transaction id
func (cp *CP) TransactionID() int {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.txnId
}

func (cp *CP) Status() (api.ChargeStatus, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := api.StatusNone

	if time.Since(cp.updated) > cp.timeout {
		return res, api.ErrTimeout
	}

	if cp.status.ErrorCode != core.NoError {
		return res, fmt.Errorf("%s: %s", cp.status.ErrorCode, cp.status.Info)
	}

	switch cp.status.Status {
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
		return api.StatusF, fmt.Errorf("chargepoint status: %s", cp.status.ErrorCode)
	default:
		return api.StatusNone, fmt.Errorf("invalid chargepoint status: %s", cp.status.Status)
	}

	return res, nil
}

var _ api.Meter = (*CP)(nil)

func (cp *CP) CurrentPower() (float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.timeout > 0 && time.Since(cp.meterUpdated) > cp.timeout {
		return 0, api.ErrNotAvailable
	}

	if m, ok := cp.measurements[string(types.MeasurandPowerActiveImport)]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit), err
	}

	return 0, api.ErrNotAvailable
}

var _ api.MeterEnergy = (*CP)(nil)

func (cp *CP) TotalEnergy() (float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.timeout > 0 && time.Since(cp.meterUpdated) > cp.timeout {
		return 0, api.ErrNotAvailable
	}

	if m, ok := cp.measurements[string(types.MeasurandEnergyActiveImportRegister)]; ok {
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

var _ api.PhaseCurrents = (*CP)(nil)

func (cp *CP) Currents() (float64, float64, float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.timeout > 0 && time.Since(cp.meterUpdated) > cp.timeout {
		return 0, 0, 0, api.ErrNotAvailable
	}

	currents := make([]float64, 0, 3)

	for phase := 1; phase <= 3; phase++ {
		m, ok := cp.measurements[getKeyCurrentPhase(phase)]
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
