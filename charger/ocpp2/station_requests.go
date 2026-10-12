package ocpp2

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/remotecontrol"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// SetChargingProfileRequest sendet ein ChargingProfile an die Station.
func (st *Station) SetChargingProfileRequest(evseId int, profile types.ChargingProfile) error {
	if !st.Connected() {
		return api.ErrTimeout
	}
	rc := make(chan error, 1)
	err := st.cs.SetChargingProfile(
		st.id,
		func(resp *smartcharging.SetChargingProfileResponse, err error) {
			if err == nil && resp != nil && resp.Status != smartcharging.ChargingProfileStatusAccepted {
				err = errors.New(string(resp.Status))
			}
			rc <- err
		},
		evseId,
		&profile,
	)
	return wait(err, rc)
}

// RequestStartTransactionRequest fordert die Station auf, eine Transaktion zu starten.
func (st *Station) RequestStartTransactionRequest(evseId int, idToken types.IdToken) error {
	if !st.Connected() {
		return api.ErrTimeout
	}
	rc := make(chan error, 1)
	err := st.cs.RequestStartTransaction(
		st.id,
		func(resp *remotecontrol.RequestStartTransactionResponse, err error) {
			if err == nil && resp != nil && resp.Status != remotecontrol.RequestStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}
			rc <- err
		},
		0,
		idToken,
		func(req *remotecontrol.RequestStartTransactionRequest) {
			req.EvseID = &evseId
		},
	)
	return wait(err, rc)
}

// RequestStopTransactionRequest fordert die Station auf, eine Transaktion zu beenden.
func (st *Station) RequestStopTransactionRequest(transactionId string) error {
	if !st.Connected() {
		return api.ErrTimeout
	}
	rc := make(chan error, 1)
	err := st.cs.RequestStopTransaction(
		st.id,
		func(resp *remotecontrol.RequestStopTransactionResponse, err error) {
			if err == nil && resp != nil && resp.Status != remotecontrol.RequestStartStopStatusAccepted {
				err = errors.New(string(resp.Status))
			}
			rc <- err
		},
		transactionId,
	)
	return wait(err, rc)
}

// ChangeAvailabilityRequest ändert den Betriebsstatus einer EVSE.
func (st *Station) ChangeAvailabilityRequest(evseId int, status availability.OperationalStatus) error {
	if !st.Connected() {
		return api.ErrTimeout
	}
	rc := make(chan error, 1)
	err := st.cs.ChangeAvailability(
		st.id,
		func(resp *availability.ChangeAvailabilityResponse, err error) {
			if err == nil && resp != nil &&
				resp.Status != availability.ChangeAvailabilityStatusAccepted &&
				resp.Status != availability.ChangeAvailabilityStatusScheduled {
				err = errors.New(string(resp.Status))
			}
			rc <- err
		},
		status,
		func(req *availability.ChangeAvailabilityRequest) {
			req.Evse = &types.EVSE{ID: evseId}
		},
	)
	return wait(err, rc)
}

// TriggerMessageRequest fordert die Station auf, eine Nachricht zu senden.
func (st *Station) TriggerMessageRequest(evseId int, message remotecontrol.MessageTrigger) error {
	if !st.Connected() {
		return api.ErrTimeout
	}
	rc := make(chan error, 1)
	err := st.cs.TriggerMessage(
		st.id,
		func(resp *remotecontrol.TriggerMessageResponse, err error) {
			if err == nil && resp != nil && resp.Status != remotecontrol.TriggerMessageStatusAccepted {
				err = errors.New(string(resp.Status))
			}
			rc <- err
		},
		message,
		func(req *remotecontrol.TriggerMessageRequest) {
			if evseId > 0 {
				req.Evse = &types.EVSE{ID: evseId}
			}
		},
	)
	return wait(err, rc)
}