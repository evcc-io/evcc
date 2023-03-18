package charger

import (
	"fmt"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type ChargePointHandler struct {
	triggerC chan remotetrigger.MessageTrigger
}

func (handler *ChargePointHandler) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (confirmation *core.ChangeAvailabilityConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusAccepted), nil
}

func (handler *ChargePointHandler) OnChangeConfiguration(request *core.ChangeConfigurationRequest) (confirmation *core.ChangeConfigurationConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusAccepted), nil
}

func (handler *ChargePointHandler) OnClearCache(request *core.ClearCacheRequest) (confirmation *core.ClearCacheConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewClearCacheConfirmation(core.ClearCacheStatusAccepted), nil
}

func (handler *ChargePointHandler) OnDataTransfer(request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewDataTransferConfirmation(core.DataTransferStatusAccepted), nil
}

func (handler *ChargePointHandler) OnGetConfiguration(request *core.GetConfigurationRequest) (confirmation *core.GetConfigurationConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	one := "1"
	return core.NewGetConfigurationConfirmation([]core.ConfigurationKey{
		{Key: "AuthorizationKey"},
		{Key: "NumberOfConnectors", Value: &one},
		{Key: "ChargeProfileMaxStackLevel", Value: &one},
		{Key: "ChargingScheduleMaxPeriods", Value: &one},
		{Key: "MaxChargingProfilesInstalled", Value: &one},
		{Key: "ChargingScheduleAllowedChargingRateUnit", Value: &one},
	}), nil
}

func (handler *ChargePointHandler) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (confirmation *core.RemoteStartTransactionConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

func (handler *ChargePointHandler) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (confirmation *core.RemoteStopTransactionConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

func (handler *ChargePointHandler) OnReset(request *core.ResetRequest) (confirmation *core.ResetConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewResetConfirmation(core.ResetStatusAccepted), nil
}

func (handler *ChargePointHandler) OnUnlockConnector(request *core.UnlockConnectorRequest) (confirmation *core.UnlockConnectorConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusUnlocked), nil
}

func (handler *ChargePointHandler) OnTriggerMessage(request *remotetrigger.TriggerMessageRequest) (confirmation *remotetrigger.TriggerMessageConfirmation, err error) {
	fmt.Printf("%T %+v\n", request, request)

	if c := handler.triggerC; request != nil && c != nil {
		select {
		case c <- request.RequestedMessage:
		default:
		}
	}

	return remotetrigger.NewTriggerMessageConfirmation(remotetrigger.TriggerMessageStatusAccepted), nil
}
