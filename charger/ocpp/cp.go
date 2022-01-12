package ocpp

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

const timeout = time.Minute

// txnCount is the global transaction id counter
var txnCount int64

type CP struct {
	mu  sync.Mutex
	log *util.Logger
	id  string
	txn int // current transaction

	updated     time.Time
	initialized *sync.Cond
	boot        *core.BootNotificationRequest
	status      *core.StatusNotificationRequest
	meterValues *core.MeterValuesRequest

	options map[string]core.ConfigurationKey
}

func (cp *CP) SetOptions(opts []core.ConfigurationKey) {
	for _, opt := range opts {
		cp.options[opt.Key] = opt
	}
}

func (cp *CP) GetOption(key string) (core.ConfigurationKey, error) {
	opt, found := cp.options[key]
	if !found {
		return core.ConfigurationKey{}, fmt.Errorf("requested option key could not be found")
	}

	return opt, nil
}

func (cp *CP) GetNumberOfSupportedConnectors() (int, error) {
	opt, err := cp.GetOption("NumberOfConnectors")
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(*opt.Value)

}

// Boot waits for the CP to register itself
func (cp *CP) Boot() error {
	bootC := make(chan struct{})
	go func() {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		for cp.boot == nil || cp.status == nil {
			cp.initialized.Wait()
		}

		close(bootC)
	}()

	select {
	case <-bootC:
		cp.update()
		return nil
	case <-time.After(timeout):
		return api.ErrTimeout
	}
}

// TransactionID returns the current transaction id
func (cp *CP) TransactionID() int {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.txn
}

func (cp *CP) Status() (api.ChargeStatus, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := api.StatusNone

	if time.Since(cp.updated) > timeout {
		return res, api.ErrTimeout
	}

	if cp.status.ErrorCode != core.NoError {
		return res, fmt.Errorf("chargepoint error: %s", cp.status.ErrorCode)
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
		return api.StatusF, fmt.Errorf("chargepoint status: %s", cp.status.Status)
	default:
		return api.StatusNone, fmt.Errorf("invalid chargepoint status: %s", cp.status.Status)
	}

	return res, nil
}
