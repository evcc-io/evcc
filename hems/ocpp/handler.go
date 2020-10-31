package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

func updateStatus(stateHandler *core.ChargePointHandler, connector int, status core.ChargePointStatus, props ...func(request *core.StatusNotificationRequest)) {
	// if connector == 0 {
	// 	stateHandler.status = status
	// } else {
	// 	stateHandler.connectors[connector].status = status
	// }

	// statusConfirmation, err := chargePoint.StatusNotification(connector, stateHandler.errorCode, status, props...)
	// // checkError(err)

	// if connector == 0 {
	// 	// logDefault(statusConfirmation.GetFeatureName()).Infof("status for all connectors updated to %v", status)
	// } else {
	// 	// logDefault(statusConfirmation.GetFeatureName()).Infof("status for connector %v updated to %v", connector, status)
	// }
}

func (s *OCPP) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (confirmation *core.ChangeAvailabilityConfirmation, err error) {
	if _, ok := s.connectors[request.ConnectorId]; !ok {
		s.log.TRACE.Printf("%s: cannot change availability for invalid connector %v", request.ConnectorId, request.GetFeatureName())
		return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusRejected), nil
	}

	s.connectors[request.ConnectorId].availability = request.Type
	if request.Type == core.AvailabilityTypeInoperative {
		// TODO: stop ongoing transactions
		s.connectors[request.ConnectorId].status = core.ChargePointStatusUnavailable
	} else {
		s.connectors[request.ConnectorId].status = core.ChargePointStatusAvailable
	}

	s.log.TRACE.Printf("%s: change availability for connector %v", request.GetFeatureName(), request.ConnectorId)

	// go updateStatus(s, request.ConnectorId, s.connectors[request.ConnectorId].status)

	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusAccepted), nil
}

func (s *OCPP) OnClearCache(request *core.ClearCacheRequest) (confirmation *core.ClearCacheConfirmation, err error) {
	s.log.TRACE.Printf("%s: cleared mocked cache", request.GetFeatureName())
	return core.NewClearCacheConfirmation(core.ClearCacheStatusAccepted), nil
}

func (s *OCPP) OnDataTransfer(request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	s.log.TRACE.Printf("%s: data transfer [Vendor: %v Message: %v]: %v", request.VendorId, request.MessageId, request.Data, request.GetFeatureName())
	return core.NewDataTransferConfirmation(core.DataTransferStatusAccepted), nil
}

func (s *OCPP) OnReset(request *core.ResetRequest) (confirmation *core.ResetConfirmation, err error) {
	s.log.TRACE.Printf("%s: no reset logic implemented yet", request.GetFeatureName())
	return core.NewResetConfirmation(core.ResetStatusAccepted), nil
}

func (s *OCPP) OnUnlockConnector(request *core.UnlockConnectorRequest) (confirmation *core.UnlockConnectorConfirmation, err error) {
	_, ok := s.connectors[request.ConnectorId]
	if !ok {
		s.log.TRACE.Printf("%s: couldn't unlock invalid connector %v", request.ConnectorId, request.GetFeatureName())
		return core.NewUnlockConnectorConfirmation(core.UnlockStatusNotSupported), nil
	}

	s.log.TRACE.Printf("%s: unlocked connector %v", request.ConnectorId, request.GetFeatureName())

	return core.NewUnlockConnectorConfirmation(core.UnlockStatusUnlocked), nil
}
