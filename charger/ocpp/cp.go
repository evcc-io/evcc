package ocpp

import (
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const timeout = time.Minute

type CP struct {
	mu          sync.Mutex
	log         *util.Logger
	id          string
	available   bool
	meterValues []types.MeterValue
	status      core.StatusNotificationRequest
}

func (cp *CP) SetAvailable(available bool) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.available = available
}

func (cp *CP) Available() bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.available
}

func (cp *CP) Status() (api.ChargeStatus, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	res := api.StatusNone

	if time.Since(cp.status.Timestamp.Time) > timeout {
		return res, api.ErrTimeout
	}

	if cp.status.ErrorCode != core.NoError {
		return res, fmt.Errorf("chargepoint error: %s", cp.status.ErrorCode)
	}

	res = api.StatusA

	switch cp.status.Status {
	case core.ChargePointStatusUnavailable: // "Unavailable"
		res = api.StatusA
	case core.ChargePointStatusAvailable, // "Available"
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
