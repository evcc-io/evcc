package profile

import (
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type Core struct {
	log           *util.Logger
	configuration ConfigMap
}

func NewCore(log *util.Logger, config ConfigMap) *Core {
	return &Core{
		log:           log,
		configuration: config,
	}
}

// OnChangeAvailability handles the CS message
func (s *Core) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (confirmation *core.ChangeAvailabilityConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusRejected), nil
}

// OnUnlockConnector handles the CS message
func (s *Core) OnUnlockConnector(request *core.UnlockConnectorRequest) (confirmation *core.UnlockConnectorConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusUnlocked), nil
}

// OnRemoteStartTransaction handles the CS message
func (s *Core) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (confirmation *core.RemoteStartTransactionConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}

// OnRemoteStopTransaction handles the CS message
func (s *Core) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (confirmation *core.RemoteStopTransactionConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}
