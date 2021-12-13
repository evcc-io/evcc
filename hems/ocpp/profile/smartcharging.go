package profile

import (
	"github.com/evcc-io/evcc/util/logx"
	"github.com/go-kit/log/level"
	sc "github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
)

type SmartCharging struct {
	log logx.Logger
}

func NewSmartCharging(log logx.Logger) *SmartCharging {
	return &SmartCharging{
		log: level.Debug(log),
	}
}

// OnSetChargingProfile handles the CS message
func (s *SmartCharging) OnSetChargingProfile(request *sc.SetChargingProfileRequest) (confirmation *sc.SetChargingProfileConfirmation, err error) {
	_ = s.log.Log("feature", request.GetFeatureName(), "recv", request)
	return sc.NewSetChargingProfileConfirmation(sc.ChargingProfileStatusRejected), nil
}

// OnClearChargingProfile handles the CS message
func (s *SmartCharging) OnClearChargingProfile(request *sc.ClearChargingProfileRequest) (confirmation *sc.ClearChargingProfileConfirmation, err error) {
	_ = s.log.Log("feature", request.GetFeatureName(), "recv", request)
	return sc.NewClearChargingProfileConfirmation(sc.ClearChargingProfileStatusUnknown), nil
}

// OnGetCompositeSchedule handles the CS message
func (s *SmartCharging) OnGetCompositeSchedule(request *sc.GetCompositeScheduleRequest) (confirmation *sc.GetCompositeScheduleConfirmation, err error) {
	_ = s.log.Log("feature", request.GetFeatureName(), "recv", request)
	return sc.NewGetCompositeScheduleConfirmation(sc.GetCompositeScheduleStatusRejected), nil
}
