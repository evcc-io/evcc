package profile

import (
	"github.com/evcc-io/evcc/util/logx"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type Core struct {
	log           logx.Logger
	configuration ConfigMap
}

func NewCore(log logx.Logger, config ConfigMap) *Core {
	return &Core{
		log:           logx.TraceLevel(log),
		configuration: config,
	}
}

// OnChangeAvailability handles the CS message
func (s *Core) OnChangeAvailability(request *core.ChangeAvailabilityRequest) (confirmation *core.ChangeAvailabilityConfirmation, err error) {
	_ = s.log.Log("feature", request.GetFeatureName(), "recv", request)
	return core.NewChangeAvailabilityConfirmation(core.AvailabilityStatusRejected), nil
}

// OnUnlockConnector handles the CS message
func (s *Core) OnUnlockConnector(request *core.UnlockConnectorRequest) (confirmation *core.UnlockConnectorConfirmation, err error) {
	_ = s.log.Log("feature", request.GetFeatureName(), "recv", request)
	return core.NewUnlockConnectorConfirmation(core.UnlockStatusUnlocked), nil
}

// OnRemoteStartTransaction handles the CS message
func (s *Core) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (confirmation *core.RemoteStartTransactionConfirmation, err error) {
	_ = s.log.Log("feature", request.GetFeatureName(), "recv", request)
	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}

// OnRemoteStopTransaction handles the CS message
func (s *Core) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (confirmation *core.RemoteStopTransactionConfirmation, err error) {
	_ = s.log.Log("feature", request.GetFeatureName(), "recv", request)
	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}
