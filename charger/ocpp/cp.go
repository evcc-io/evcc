package ocpp

import (
	"fmt"
	// "strconv"
	// "strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
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
	mu      sync.Mutex
	log     *util.Logger
	once    sync.Once

	id string
	details *core.BootNotificationRequest

	updated time.Time
	timeout time.Duration

	connectC, bootC, statusC chan struct{} // signals

	statusUpdated     time.Time
	status            *core.StatusNotificationRequest

	meterUpdated time.Time
	measurements map[string]types.SampledValue

	transactions map[int]*Transaction
	currentTransaction *Transaction
	lastTransactionId int
}

func NewChargePoint(log *util.Logger, id string, timeout time.Duration) *CP {
	return &CP{
		log:          log,
		id:           id,
		connectC:     make(chan struct{}),
		bootC:        make(chan struct{}),
		statusC:      make(chan struct{}),
		measurements: make(map[string]types.SampledValue),
		timeout:      timeout,
		lastTransactionId: 0,
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

func (cp *CP) HasBooted() <-chan struct{} {
	return cp.bootC
}

func (cp *CP) Reset() <-chan error {
	rc := make(chan error, 1)
	Instance().Reset(cp.ID(), func(request *core.ResetConfirmation, err error) {
		cp.log.TRACE.Printf("%T: %+v", request, request)

		if request != nil && request.Status != core.ResetStatusAccepted{
			cp.log.ERROR.Printf("chargepoint rejected Reset request")
		}

		rc <- err
	}, core.ResetTypeSoft)

	return rc
}

func (cp *CP) TriggerBootNotification() <-chan error {
	rc := make(chan error, 1)
	Instance().TriggerMessage(cp.ID(), func(request *remotetrigger.TriggerMessageConfirmation, err error) {
		cp.log.TRACE.Printf("%T: %+v", request, request)

		if request != nil && request.Status != remotetrigger.TriggerMessageStatusAccepted{
			cp.log.ERROR.Printf("chargepoint rejected BootNotification trigger")
		}

		rc <- err
	}, core.BootNotificationFeatureName)

	return rc
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
		return true
	case <-time.After(timeout):
		return false
	}
}

// transaction related methods

func (cp *CP) InitTransaction() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.currentTransaction = NewTransaction(cp.lastTransactionId+1)
	// save transaction Id for later
	cp.lastTransactionId = cp.currentTransaction.Id
}


func (cp *CP) FinishTransaction() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.currentTransaction.SetStatus(TransactionFinished)
	cp.currentTransaction = nil
}

// TransactionID returns the current transaction id
func (cp *CP) TransactionID() int {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.currentTransaction.Id
}

func (cp *CP) CurrentTransaction() *Transaction {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.currentTransaction
}

func (cp *CP) HasTransaction() bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.currentTransaction != nil
}

func (cp *CP) SetTransactionStatus(status TransactionState) {
	if cp.currentTransaction != nil {
		cp.currentTransaction.SetStatus(status)
	}
}

func (cp *CP) GetTransactionStatus() TransactionState{
	if cp.currentTransaction == nil {
		return TransactionUndefined
	}
	return cp.currentTransaction.Status()
}

func (cp *CP) Enabled() (bool, error) {
	if !cp.HasTransaction() {
		return false, nil
	}

	switch cp.GetTransactionStatus() {
	case TransactionStarting, TransactionRunning, TransactionSuspended:
		return true, nil
	default:
		return false, nil
	}
}

func (cp *CP) Status() (api.ChargeStatus, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := api.StatusNone

	if time.Since(cp.statusUpdated) > cp.timeout {
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

	// state correction based on ongoing transaction
	if cp.currentTransaction == nil && cp.status.Status == core.ChargePointStatusPreparing {
		res = api.StatusA
	}

	return res, nil
}

