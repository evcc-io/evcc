package ocpp

import (
	"errors"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

func (cp *CP) ChangeAvailabilityRequest(connectorId int, availabilityType core.AvailabilityType) error {
	rc := make(chan error, 1)

	err := Instance().ChangeAvailability(cp.id, func(request *core.ChangeAvailabilityConfirmation, err error) {
		if err == nil && request != nil && request.Status != core.AvailabilityStatusAccepted && request.Status != core.AvailabilityStatusScheduled {
			err = errors.New(string(request.Status))
		}

		rc <- err
	}, connectorId, availabilityType)

	return wait(err, rc)
}

func (cp *CP) GetCompositeScheduleRequest(connectorId int, duration int) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	var res *smartcharging.GetCompositeScheduleConfirmation
	rc := make(chan error, 1)

	err := Instance().GetCompositeSchedule(cp.id, func(request *smartcharging.GetCompositeScheduleConfirmation, err error) {
		if err == nil && request != nil && request.Status != smartcharging.GetCompositeScheduleStatusAccepted {
			err = errors.New(string(request.Status))
		}

		res = request

		rc <- err
	}, connectorId, duration)

	return res, wait(err, rc)
}

func (cp *CP) RemoteStartTransactionRequest(connectorId int, idTag string) error {
	rc := make(chan error, 1)
	err := Instance().RemoteStartTransaction(cp.id, func(request *core.RemoteStartTransactionConfirmation, err error) {
		if err == nil && request != nil && request.Status != types.RemoteStartStopStatusAccepted {
			err = errors.New(string(request.Status))
		}

		rc <- err
	}, idTag, func(request *core.RemoteStartTransactionRequest) {
		if connectorId > 0 {
			request.ConnectorId = &connectorId
		}
	})

	return wait(err, rc)
}

func (cp *CP) SetChargingProfileRequest(connectorId int, profile *types.ChargingProfile) error {
	rc := make(chan error, 1)

	err := Instance().SetChargingProfile(cp.id, func(request *smartcharging.SetChargingProfileConfirmation, err error) {
		if err == nil && request != nil && request.Status != smartcharging.ChargingProfileStatusAccepted {
			err = errors.New(string(request.Status))
		}

		rc <- err
	}, connectorId, profile)

	return wait(err, rc)
}

func (cp *CP) TriggerMessageRequest(connectorId int, requestedMessage remotetrigger.MessageTrigger) error {
	rc := make(chan error, 1)

	err := Instance().TriggerMessage(cp.id, func(request *remotetrigger.TriggerMessageConfirmation, err error) {
		if err == nil && request != nil && request.Status != remotetrigger.TriggerMessageStatusAccepted {
			err = errors.New(string(request.Status))
		}

		rc <- err
	}, requestedMessage, func(request *remotetrigger.TriggerMessageRequest) {
		if connectorId > 0 {
			request.ConnectorId = &connectorId
		}
	})

	return wait(err, rc)
}

func (cp *CP) ChangeConfigurationRequest(key, value string) error {
	rc := make(chan error, 1)

	err := Instance().ChangeConfiguration(cp.id, func(request *core.ChangeConfigurationConfirmation, err error) {
		if err == nil && request != nil && request.Status != core.ConfigurationStatusAccepted {
			err = errors.New(string(request.Status))
		}

		rc <- err
	}, key, value)

	return wait(err, rc)
}

func (cp *CP) GetConfigurationRequest() (*core.GetConfigurationConfirmation, error) {
	rc := make(chan error, 1)

	var res *core.GetConfigurationConfirmation
	err := Instance().GetConfiguration(cp.id, func(request *core.GetConfigurationConfirmation, err error) {
		res = request

		rc <- err
	}, nil)

	return res, wait(err, rc)
}
