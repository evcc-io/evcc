package ocpp

import (
	core "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
)

func (cs *CS) OnAuthorize(chargePointId string, request *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.Authorize(request)
}

func (cs *CS) OnBootNotification(chargePointId string, request *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.BootNotification(request)
}

func (cs *CS) OnDataTransfer(chargePointId string, request *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.DataTransfer(request)
}

func (cs *CS) OnHeartbeat(chargePointId string, request *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.Heartbeat(request)
}

func (cs *CS) TriggerMeterValueRequest(cp *CP) {
	callback := func(request *remotetrigger.TriggerMessageConfirmation, err error) {
		cs.log.TRACE.Printf("TriggerMessageRequest %T: %+v", request, request)
	}

	if err := cs.TriggerMessage(cp.id, callback, core.MeterValuesFeatureName); err != nil {
		cs.log.DEBUG.Printf("failed sending TriggerMessageRequest: %s", err)
	}
}

func (cs *CS) OnMeterValues(chargePointId string, request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	conf, err := cp.MeterValues(request)
	// if request != nil {
	// 	cs.log.DEBUG.Printf("%s < %s", request.MeterValue[0].Timestamp, time.Now().Add(-9*time.Second))
	// 	if request.MeterValue[0].Timestamp.Before(time.Now().Add(-9 * time.Second)) {

	// 		go func() {
	// 			cs.log.DEBUG.Printf("send extra meter value request to catch up")
	// 			time.Sleep(3 * time.Second)
	// 			cs.TriggerMeterValueRequest(cp)
	// 		}()
	// 	}
	// }

	return conf, err
}

func (cs *CS) OnStatusNotification(chargePointId string, request *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StatusNotification(request)
}

func (cs *CS) OnStartTransaction(chargePointId string, request *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StartTransaction(request)
}

func (cs *CS) OnStopTransaction(chargePointId string, request *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.StopTransaction(request)
}

func (cs *CS) OnDiagnosticsStatusNotification(chargePointId string, request *firmware.DiagnosticsStatusNotificationRequest) (confirmation *firmware.DiagnosticsStatusNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.DiagnosticStatusNotification(request)
}

func (cs *CS) OnFirmwareStatusNotification(chargePointId string, request *firmware.FirmwareStatusNotificationRequest) (confirmation *firmware.FirmwareStatusNotificationConfirmation, err error) {
	cp, err := cs.chargepointByID(chargePointId)
	if err != nil {
		return nil, err
	}

	return cp.FirmwareStatusNotification(request)
}

func (cs *CS) TriggerResetRequest(cp *CP, resetType core.ResetType) {
	callback := func(request *core.ResetConfirmation, err error) {
		cs.log.TRACE.Printf("TriggerResetRequest %T: %+v", request, request)
	}

	if err := cs.Reset(cp.id, callback, resetType); err != nil {
		cs.log.DEBUG.Printf("failed sending TriggerResetRequest: %s", err)
	}
}
