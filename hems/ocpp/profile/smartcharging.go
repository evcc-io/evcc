package profile

import (
	"github.com/evcc-io/evcc/util/log"
	sc "github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
)

type SmartCharging struct {
	log log.Logger
}

func NewSmartCharging(log log.Logger) *SmartCharging {
	return &SmartCharging{
		log: log,
	}
}

// OnSetChargingProfile handles the CS message
func (s *SmartCharging) OnSetChargingProfile(request *sc.SetChargingProfileRequest) (confirmation *sc.SetChargingProfileConfirmation, err error) {
	s.log.Trace("recv: %s %+v", request.GetFeatureName(), request)
	return sc.NewSetChargingProfileConfirmation(sc.ChargingProfileStatusRejected), nil
}

// OnClearChargingProfile handles the CS message
func (s *SmartCharging) OnClearChargingProfile(request *sc.ClearChargingProfileRequest) (confirmation *sc.ClearChargingProfileConfirmation, err error) {
	s.log.Trace("recv: %s %+v", request.GetFeatureName(), request)
	return sc.NewClearChargingProfileConfirmation(sc.ClearChargingProfileStatusUnknown), nil
}

// OnGetCompositeSchedule handles the CS message
func (s *SmartCharging) OnGetCompositeSchedule(request *sc.GetCompositeScheduleRequest) (confirmation *sc.GetCompositeScheduleConfirmation, err error) {
	s.log.Trace("recv: %s %+v", request.GetFeatureName(), request)
	return sc.NewGetCompositeScheduleConfirmation(sc.GetCompositeScheduleStatusRejected), nil
}
