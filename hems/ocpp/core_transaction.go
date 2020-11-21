package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// OnChangeAvailability handles the CS message
func (s *OCPP) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (confirmation *core.ChangeAvailabilityConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusRejected), nil
}

// OnUnlockConnector handles the CS message
func (s *OCPP) OnUnlockConnector(request *core.UnlockConnectorRequest) (confirmation *core.UnlockConnectorConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusUnlocked), nil
}

// OnRemoteStartTransaction handles the CS message
func (s *OCPP) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (confirmation *core.RemoteStartTransactionConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}

// OnRemoteStopTransaction handles the CS message
func (s *OCPP) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (confirmation *core.RemoteStopTransactionConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}
