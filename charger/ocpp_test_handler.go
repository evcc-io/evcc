package charger

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type ChargePointHandler struct {
	triggerC chan remotetrigger.MessageTrigger
}

// core

func (handler *ChargePointHandler) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (confirmation *core.ChangeAvailabilityConfirmation, err error) {
	defer func() { handler.triggerC <- core.ChangeAvailabilityFeatureName }()
	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusAccepted), nil
}

func (handler *ChargePointHandler) OnChangeConfiguration(request *core.ChangeConfigurationRequest) (confirmation *core.ChangeConfigurationConfirmation, err error) {
	return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusAccepted), nil
}

func (handler *ChargePointHandler) OnClearCache(request *core.ClearCacheRequest) (confirmation *core.ClearCacheConfirmation, err error) {
	return core.NewClearCacheConfirmation(core.ClearCacheStatusAccepted), nil
}

func (handler *ChargePointHandler) OnDataTransfer(request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	return core.NewDataTransferConfirmation(core.DataTransferStatusAccepted), nil
}

func (handler *ChargePointHandler) OnGetConfiguration(request *core.GetConfigurationRequest) (confirmation *core.GetConfigurationConfirmation, err error) {
	one := "1"
	meter := "Power.Active.Import,Energy.Active.Import.Register"
	return core.NewGetConfigurationConfirmation([]core.ConfigurationKey{
		{Key: "AuthorizationKey"},
		{Key: "NumberOfConnectors", Value: &one},
		{Key: "ChargeProfileMaxStackLevel", Value: &one},
		{Key: "ChargingScheduleMaxPeriods", Value: &one},
		{Key: "MaxChargingProfilesInstalled", Value: &one},
		{Key: "ChargingScheduleAllowedChargingRateUnit", Value: &one},
		{Key: "MeterValuesSampledData", Value: &meter},
	}), nil
}

func (handler *ChargePointHandler) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (confirmation *core.RemoteStartTransactionConfirmation, err error) {
	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

func (handler *ChargePointHandler) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (confirmation *core.RemoteStopTransactionConfirmation, err error) {
	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
}

func (handler *ChargePointHandler) OnReset(request *core.ResetRequest) (confirmation *core.ResetConfirmation, err error) {
	return core.NewResetConfirmation(core.ResetStatusAccepted), nil
}

func (handler *ChargePointHandler) OnUnlockConnector(request *core.UnlockConnectorRequest) (confirmation *core.UnlockConnectorConfirmation, err error) {
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusUnlocked), nil
}

func (handler *ChargePointHandler) OnTriggerMessage(request *remotetrigger.TriggerMessageRequest) (confirmation *remotetrigger.TriggerMessageConfirmation, err error) {
	defer func() { handler.triggerC <- request.RequestedMessage }()
	return remotetrigger.NewTriggerMessageConfirmation(remotetrigger.TriggerMessageStatusAccepted), nil
}

// smart charging

func (handler *ChargePointHandler) OnSetChargingProfile(request *smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileConfirmation, error) {
	return smartcharging.NewSetChargingProfileConfirmation(smartcharging.ChargingProfileStatusAccepted), nil
}

func (handler *ChargePointHandler) OnClearChargingProfile(request *smartcharging.ClearChargingProfileRequest) (*smartcharging.ClearChargingProfileConfirmation, error) {
	return smartcharging.NewClearChargingProfileConfirmation(smartcharging.ClearChargingProfileStatusAccepted), nil
}

func (handler *ChargePointHandler) OnGetCompositeSchedule(request *smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	return smartcharging.NewGetCompositeScheduleConfirmation(smartcharging.GetCompositeScheduleStatusAccepted), nil
}
