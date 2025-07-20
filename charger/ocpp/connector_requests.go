package ocpp

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func (conn *Connector) ChangeAvailabilityRequest(availabilityType core.AvailabilityType) error {
	return conn.cp.ChangeAvailabilityRequest(conn.id, availabilityType)
}

func (conn *Connector) GetCompositeScheduleRequest(duration int) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	return conn.cp.GetCompositeScheduleRequest(conn.id, duration)
}

func (conn *Connector) RemoteStartTransactionRequest(idTag string) error {
	return conn.cp.RemoteStartTransactionRequest(conn.id, idTag)
}

func (conn *Connector) SetChargingProfileRequest(profile *types.ChargingProfile) error {
	return conn.cp.SetChargingProfileRequest(conn.id, profile)
}

func (conn *Connector) TriggerMessageRequest(requestedMessage remotetrigger.MessageTrigger) error {
	return conn.cp.TriggerMessageRequest(conn.id, requestedMessage)
}
