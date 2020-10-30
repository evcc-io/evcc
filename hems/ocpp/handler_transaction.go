package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func (s *OCPP) OnRemoteStartTransaction(request *core.RemoteStartTransactionRequest) (confirmation *core.RemoteStartTransactionConfirmation, err error) {
	if request.ConnectorId != nil {
		connector, ok := s.connectors[*request.ConnectorId]
		if !ok {
			return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
		} else if connector.availability != core.AvailabilityTypeOperative || connector.status != core.ChargePointStatusAvailable || connector.currentTransaction > 0 {
			return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
		}

		s.log.TRACE.Printf("%s: started transaction %v on connector %v", connector.currentTransaction, request.GetFeatureName(), request.ConnectorId)

		connector.currentTransaction = *request.ConnectorId
		return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
	}

	s.log.TRACE.Printf("%s: couldn't start a transaction for %v without a connectorID", request.GetFeatureName(), request.IdTag)

	return core.NewRemoteStartTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}

func (s *OCPP) OnRemoteStopTransaction(request *core.RemoteStopTransactionRequest) (confirmation *core.RemoteStopTransactionConfirmation, err error) {
	for key, val := range s.connectors {
		if val.currentTransaction == request.TransactionId {
			s.log.TRACE.Printf("%s: stopped transaction %v on connector %v", request.GetFeatureName(), val.currentTransaction, key)

			val.currentTransaction = 0
			val.currentReservation = 0
			val.status = core.ChargePointStatusAvailable

			return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusAccepted), nil
		}
	}

	s.log.TRACE.Printf("%s: couldn't stop transaction %v, no such transaction is ongoing", request.GetFeatureName(), request.TransactionId)

	return core.NewRemoteStopTransactionConfirmation(types.RemoteStartStopStatusRejected), nil
}
