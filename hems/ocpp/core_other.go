package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

// OnClearCache handles the CS message
func (s *OCPP) OnClearCache(request *core.ClearCacheRequest) (confirmation *core.ClearCacheConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s", request.GetFeatureName())
	return core.NewClearCacheConfirmation(core.ClearCacheStatusAccepted), nil
}

// OnDataTransfer handles the CS message
func (s *OCPP) OnDataTransfer(request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewDataTransferConfirmation(core.DataTransferStatusAccepted), nil
}

// OnReset handles the CS message
func (s *OCPP) OnReset(request *core.ResetRequest) (confirmation *core.ResetConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewResetConfirmation(core.ResetStatusAccepted), nil
}
