package charger

import (
	"errors"

	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/authorization"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/data"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/diagnostics"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/display"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/iso15118"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/remotecontrol"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/security"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/tariffcost"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// chargingStation20Handler implements every CS-side handler interface so it
// can be plugged into ocpp-go's NewChargingStation. Most methods are stubs;
// only OnGetVariables, OnSetChargingProfile and OnTriggerMessage participate
// in the e2e flow.
type chargingStation20Handler struct {
	triggerC      chan remotecontrol.MessageTrigger
	phaseSwitchOK bool // value reported for ACPhaseSwitchingSupported
}

// availability.ChargingStationHandler

func (h *chargingStation20Handler) OnChangeAvailability(req *availability.ChangeAvailabilityRequest) (*availability.ChangeAvailabilityResponse, error) {
	return &availability.ChangeAvailabilityResponse{Status: availability.ChangeAvailabilityStatusAccepted}, nil
}

// provisioning.ChargingStationHandler

func (h *chargingStation20Handler) OnGetBaseReport(req *provisioning.GetBaseReportRequest) (*provisioning.GetBaseReportResponse, error) {
	return &provisioning.GetBaseReportResponse{Status: types.GenericDeviceModelStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetReport(req *provisioning.GetReportRequest) (*provisioning.GetReportResponse, error) {
	return &provisioning.GetReportResponse{Status: types.GenericDeviceModelStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetVariables(req *provisioning.GetVariablesRequest) (*provisioning.GetVariablesResponse, error) {
	val := "false"
	if h.phaseSwitchOK {
		val = "true"
	}

	results := make([]provisioning.GetVariableResult, 0, len(req.GetVariableData))
	for _, q := range req.GetVariableData {
		r := provisioning.GetVariableResult{
			Component:       q.Component,
			Variable:        q.Variable,
			AttributeStatus: provisioning.GetVariableStatusAccepted,
		}
		if q.Variable.Name == "ACPhaseSwitchingSupported" {
			r.AttributeValue = val
		} else {
			r.AttributeStatus = provisioning.GetVariableStatusUnknownVariable
		}
		results = append(results, r)
	}
	return &provisioning.GetVariablesResponse{GetVariableResult: results}, nil
}

func (h *chargingStation20Handler) OnReset(req *provisioning.ResetRequest) (*provisioning.ResetResponse, error) {
	return &provisioning.ResetResponse{Status: provisioning.ResetStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnSetNetworkProfile(req *provisioning.SetNetworkProfileRequest) (*provisioning.SetNetworkProfileResponse, error) {
	return &provisioning.SetNetworkProfileResponse{Status: provisioning.SetNetworkProfileStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnSetVariables(req *provisioning.SetVariablesRequest) (*provisioning.SetVariablesResponse, error) {
	return &provisioning.SetVariablesResponse{}, nil
}

// authorization.ChargingStationHandler — currently has no methods (CS does not receive on this profile)

// transactions.ChargingStationHandler

func (h *chargingStation20Handler) OnGetTransactionStatus(req *transactions.GetTransactionStatusRequest) (*transactions.GetTransactionStatusResponse, error) {
	return &transactions.GetTransactionStatusResponse{}, nil
}

// remotecontrol.ChargingStationHandler

func (h *chargingStation20Handler) OnRequestStartTransaction(req *remotecontrol.RequestStartTransactionRequest) (*remotecontrol.RequestStartTransactionResponse, error) {
	return &remotecontrol.RequestStartTransactionResponse{Status: remotecontrol.RequestStartStopStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnRequestStopTransaction(req *remotecontrol.RequestStopTransactionRequest) (*remotecontrol.RequestStopTransactionResponse, error) {
	return &remotecontrol.RequestStopTransactionResponse{Status: remotecontrol.RequestStartStopStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnTriggerMessage(req *remotecontrol.TriggerMessageRequest) (*remotecontrol.TriggerMessageResponse, error) {
	if h.triggerC != nil {
		select {
		case h.triggerC <- req.RequestedMessage:
		default:
		}
	}
	return &remotecontrol.TriggerMessageResponse{Status: remotecontrol.TriggerMessageStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnUnlockConnector(req *remotecontrol.UnlockConnectorRequest) (*remotecontrol.UnlockConnectorResponse, error) {
	return &remotecontrol.UnlockConnectorResponse{Status: remotecontrol.UnlockStatusUnlocked}, nil
}

// reservation.ChargingStationHandler

func (h *chargingStation20Handler) OnCancelReservation(req *reservation.CancelReservationRequest) (*reservation.CancelReservationResponse, error) {
	return &reservation.CancelReservationResponse{Status: reservation.CancelReservationStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnReserveNow(req *reservation.ReserveNowRequest) (*reservation.ReserveNowResponse, error) {
	return &reservation.ReserveNowResponse{Status: reservation.ReserveNowStatusAccepted}, nil
}

// localauth.ChargingStationHandler

func (h *chargingStation20Handler) OnGetLocalListVersion(req *localauth.GetLocalListVersionRequest) (*localauth.GetLocalListVersionResponse, error) {
	return &localauth.GetLocalListVersionResponse{VersionNumber: 0}, nil
}

func (h *chargingStation20Handler) OnSendLocalList(req *localauth.SendLocalListRequest) (*localauth.SendLocalListResponse, error) {
	return &localauth.SendLocalListResponse{Status: localauth.SendLocalListStatusAccepted}, nil
}

// security.ChargingStationHandler

func (h *chargingStation20Handler) OnCertificateSigned(req *security.CertificateSignedRequest) (*security.CertificateSignedResponse, error) {
	return &security.CertificateSignedResponse{Status: security.CertificateSignedStatusAccepted}, nil
}

// smartcharging.ChargingStationHandler

func (h *chargingStation20Handler) OnClearChargingProfile(req *smartcharging.ClearChargingProfileRequest) (*smartcharging.ClearChargingProfileResponse, error) {
	return &smartcharging.ClearChargingProfileResponse{Status: smartcharging.ClearChargingProfileStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetChargingProfiles(req *smartcharging.GetChargingProfilesRequest) (*smartcharging.GetChargingProfilesResponse, error) {
	return &smartcharging.GetChargingProfilesResponse{Status: smartcharging.GetChargingProfileStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetCompositeSchedule(req *smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleResponse, error) {
	return &smartcharging.GetCompositeScheduleResponse{Status: smartcharging.GetCompositeScheduleStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnSetChargingProfile(req *smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileResponse, error) {
	return &smartcharging.SetChargingProfileResponse{Status: smartcharging.ChargingProfileStatusAccepted}, nil
}

// firmware.ChargingStationHandler

func (h *chargingStation20Handler) OnPublishFirmware(req *firmware.PublishFirmwareRequest) (*firmware.PublishFirmwareResponse, error) {
	return &firmware.PublishFirmwareResponse{Status: types.GenericStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnUnpublishFirmware(req *firmware.UnpublishFirmwareRequest) (*firmware.UnpublishFirmwareResponse, error) {
	return &firmware.UnpublishFirmwareResponse{Status: firmware.UnpublishFirmwareStatusUnpublished}, nil
}

func (h *chargingStation20Handler) OnUpdateFirmware(req *firmware.UpdateFirmwareRequest) (*firmware.UpdateFirmwareResponse, error) {
	return &firmware.UpdateFirmwareResponse{Status: firmware.UpdateFirmwareStatusAccepted}, nil
}

// iso15118.ChargingStationHandler

func (h *chargingStation20Handler) OnDeleteCertificate(req *iso15118.DeleteCertificateRequest) (*iso15118.DeleteCertificateResponse, error) {
	return &iso15118.DeleteCertificateResponse{Status: iso15118.DeleteCertificateStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetInstalledCertificateIds(req *iso15118.GetInstalledCertificateIdsRequest) (*iso15118.GetInstalledCertificateIdsResponse, error) {
	return &iso15118.GetInstalledCertificateIdsResponse{Status: iso15118.GetInstalledCertificateStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnInstallCertificate(req *iso15118.InstallCertificateRequest) (*iso15118.InstallCertificateResponse, error) {
	return &iso15118.InstallCertificateResponse{Status: iso15118.CertificateStatusAccepted}, nil
}

// diagnostics.ChargingStationHandler

func (h *chargingStation20Handler) OnClearVariableMonitoring(req *diagnostics.ClearVariableMonitoringRequest) (*diagnostics.ClearVariableMonitoringResponse, error) {
	return &diagnostics.ClearVariableMonitoringResponse{}, nil
}

func (h *chargingStation20Handler) OnCustomerInformation(req *diagnostics.CustomerInformationRequest) (*diagnostics.CustomerInformationResponse, error) {
	return &diagnostics.CustomerInformationResponse{Status: diagnostics.CustomerInformationStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetLog(req *diagnostics.GetLogRequest) (*diagnostics.GetLogResponse, error) {
	return &diagnostics.GetLogResponse{Status: diagnostics.LogStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetMonitoringReport(req *diagnostics.GetMonitoringReportRequest) (*diagnostics.GetMonitoringReportResponse, error) {
	return &diagnostics.GetMonitoringReportResponse{Status: types.GenericDeviceModelStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnSetMonitoringBase(req *diagnostics.SetMonitoringBaseRequest) (*diagnostics.SetMonitoringBaseResponse, error) {
	return &diagnostics.SetMonitoringBaseResponse{Status: types.GenericDeviceModelStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnSetMonitoringLevel(req *diagnostics.SetMonitoringLevelRequest) (*diagnostics.SetMonitoringLevelResponse, error) {
	return &diagnostics.SetMonitoringLevelResponse{Status: types.GenericDeviceModelStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnSetVariableMonitoring(req *diagnostics.SetVariableMonitoringRequest) (*diagnostics.SetVariableMonitoringResponse, error) {
	return &diagnostics.SetVariableMonitoringResponse{}, nil
}

// display.ChargingStationHandler

func (h *chargingStation20Handler) OnClearDisplay(req *display.ClearDisplayRequest) (*display.ClearDisplayResponse, error) {
	return &display.ClearDisplayResponse{Status: display.ClearMessageStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnGetDisplayMessages(req *display.GetDisplayMessagesRequest) (*display.GetDisplayMessagesResponse, error) {
	return &display.GetDisplayMessagesResponse{Status: display.MessageStatusAccepted}, nil
}

func (h *chargingStation20Handler) OnSetDisplayMessage(req *display.SetDisplayMessageRequest) (*display.SetDisplayMessageResponse, error) {
	return &display.SetDisplayMessageResponse{Status: display.DisplayMessageStatusAccepted}, nil
}

// data.ChargingStationHandler

func (h *chargingStation20Handler) OnDataTransfer(req *data.DataTransferRequest) (*data.DataTransferResponse, error) {
	return &data.DataTransferResponse{Status: data.DataTransferStatusAccepted}, nil
}

// tariffcost.ChargingStationHandler

func (h *chargingStation20Handler) OnCostUpdated(req *tariffcost.CostUpdatedRequest) (*tariffcost.CostUpdatedResponse, error) {
	return &tariffcost.CostUpdatedResponse{}, nil
}

// silence unused-import linter for authorization (no CS-side methods at present)
var _ = authorization.AuthorizeFeatureName
var _ = errors.New
