package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// cp actions

func (cs *CS) OnAuthorize(id string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	// no cp handler

	res := &core.AuthorizeConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cs *CS) OnBootNotification(id string, request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnBootNotification(request)
	}

	res := &core.BootNotificationConfirmation{
		CurrentTime: types.Now(),
		Interval:    int(Timeout.Seconds()),
		Status:      core.RegistrationStatusPending, // not accepted during startup
	}

	return res, nil
}

func (cs *CS) OnDataTransfer(id string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	// no cp handler

	res := &core.DataTransferConfirmation{
		Status: core.DataTransferStatusAccepted,
	}

	return res, nil
}

func (cs *CS) OnHeartbeat(id string, request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	// no cp handler

	res := &core.HeartbeatConfirmation{
		CurrentTime: types.Now(),
	}

	return res, nil
}

func (cs *CS) OnMeterValues(id string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnMeterValues(request)
	}

	return new(core.MeterValuesConfirmation), nil
}

func (cs *CS) OnStatusNotification(id string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnStatusNotification(request)
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	// cache status for future cp connection
	if reg, ok := cs.regs[id]; ok {
		reg.mu.Lock()
		reg.status = request
		reg.mu.Unlock()
	}

	return new(core.StatusNotificationConfirmation), nil
}

func (cs *CS) OnStartTransaction(id string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		return cp.OnStartTransaction(request)
	}

	res := &core.StartTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}

	return res, nil
}

func (cs *CS) OnStopTransaction(id string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	if cp, err := cs.ChargepointByID(id); err == nil {
		cp.OnStopTransaction(request)
	}

	res := &core.StopTransactionConfirmation{
		IdTagInfo: &types.IdTagInfo{
			Status: types.AuthorizationStatusAccepted, // accept old pending stop message during startup
		},
	}

	return res, nil
}
