// +build eebus

package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amp-x/eebus/communication"
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/charger/eebus"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

const messagesTimeout = 10 * time.Second

type Certificate struct {
	Public  string
	Private string
}

type EEBUSCharger struct {
	*request.Helper
	ski string
}

func init() {
	registry.Add("eebus", NewEEBUSChargerFromConfig)
}

// NewEEBUSChargerFromConfig creates an EEBUS charger from generic config
func NewEEBUSChargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Ski         string
		Certificate Certificate
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEEBUSCharger(cc.Ski, cc.Certificate.Public, cc.Certificate.Private)
}

// NewMobileConnect creates MCC charger
func NewEEBUSCharger(ski, cert, key string) (*EEBUSCharger, error) {
	log := util.NewLogger("mcc")

	if eebus.Instance == nil {
		var err error

		instance, err := eebus.New(log, key, cert)
		if err != nil {
			return nil, err
		}
		eebus.Instance = instance
	}

	shortedSki := strings.ReplaceAll(ski, "-", "")

	c := &EEBUSCharger{
		Helper: request.NewHelper(log),
		ski:    shortedSki,
	}

	// on start we need to disable charging as it would otherwise start with max current
	_ = c.Enable(false)

	return c, nil
}

// Status implements the api.Charger interface
func (eeb *EEBUSCharger) Status() (api.ChargeStatus, error) {
	data, err := eebus.Instance.GetData(eeb.ski)
	if err != nil {
		return api.StatusNone, err
	}

	currentState := data.EVData.ChargeState

	switch currentState {
	case communication.EVChargeStateEnumTypeUnplugged: // Unplugged
		return api.StatusA, nil
	case communication.EVChargeStateEnumTypeFinished, communication.EVChargeStateEnumTypePaused: // Finished, Paused
		return api.StatusB, nil
	case communication.EVChargeStateEnumTypeError: // Error
		return api.StatusF, nil
	case communication.EVChargeStateEnumTypeActive: // Active
		return api.StatusC, nil
	}
	return api.StatusNone, fmt.Errorf("properties unknown result: %s", currentState)
}

// Enabled implements the api.Charger interface
func (eeb *EEBUSCharger) Enabled() (bool, error) {
	data, err := eebus.Instance.GetData(eeb.ski)
	if err != nil {
		return false, err
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
		return false, nil
	}

	// are all currents above 1?
	// if disabled, value is < 0.3, mostly 0.1
	// if current was set to min, the reported current is lower, e.g. 5.9
	if data.EVData.Measurements.CurrentL1 >= 0.3 ||
		data.EVData.Measurements.CurrentL2 >= 0.3 ||
		data.EVData.Measurements.CurrentL3 >= 0.3 {
		return true, nil
	}

	return false, nil
}

// Enable implements the api.Charger interface
func (eeb *EEBUSCharger) Enable(enable bool) error {
	data, err := eebus.Instance.GetData(eeb.ski)
	if err != nil {
		return err
	}

	if !enable {
		// Important notes on enabling/disabling!!
		// ISO15118 mode:
		//   non-asymetric or all phases set to 0: the OBC will wait for 1 minute, if the values remain after 1 min, it will pause then
		//   asymetric and only some phases set to 0: no pauses or waiting for changes required
		//   asymetric mode requires Plug & Charge (PnC) and Value Added Services (VAS)
		// IEC61851 mode:
		//   switching between 1/3 phases: stop charging, pause for 2 minutes, change phases, resume charging
		//   frequent switching should be avoided by all means!
		return eebus.Instance.SetCurrents(eeb.ski, 0.0, 0.0, 0.0)
	}

	return eebus.Instance.SetCurrents(eeb.ski, data.EVData.LimitsL1.Min, data.EVData.LimitsL2.Min, data.EVData.LimitsL3.Min)
}

// MaxCurrent implements the api.Charger interface
func (eeb *EEBUSCharger) MaxCurrent(current int64) error {
	return eeb.MaxCurrentMillis(float64(current))
}

// MaxCurrentMillis implements the api.ChargerEx interface
func (eeb *EEBUSCharger) MaxCurrentMillis(current float64) error {
	data, err := eebus.Instance.GetData(eeb.ski)
	if err != nil {
		return err
	}

	if data.EVData.LimitsL1.Min == 0 {
		return errors.New("we did not yet receive min and max currents to validate the call of MaxCurrent")
	}

	if current < data.EVData.LimitsL1.Min {
		return fmt.Errorf("value is lower than the allowed minimum value %f", data.EVData.LimitsL1.Min)
	}

	if current > data.EVData.LimitsL1.Max {
		return fmt.Errorf("value is higher than the allowed maximum value %f", data.EVData.LimitsL1.Max)
	}

	return eebus.Instance.SetCurrents(eeb.ski, current, current, current)
}

var _ api.Meter = (*EEBUSCharger)(nil)

// CurrentPower implements the api.Meter interface
func (eeb *EEBUSCharger) CurrentPower() (float64, error) {
	data, err := eebus.Instance.GetData(eeb.ski)
	if err != nil {
		return 0.0, err
	}

	power := data.EVData.Measurements.PowerL1 + data.EVData.Measurements.PowerL2 + data.EVData.Measurements.PowerL3

	return power, nil
}

var _ api.ChargeRater = (*EEBUSCharger)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (eeb *EEBUSCharger) ChargedEnergy() (float64, error) {
	data, err := eebus.Instance.GetData(eeb.ski)
	if err != nil {
		return 0.0, err
	}

	return data.EVData.Measurements.ChargedEnergy, nil
}

// var _ api.ChargeTimer = (*EEBUSCharger)(nil)

// // ChargingTime implements the api.ChargeTimer interface
// func (eeb *EEBUSCharger) ChargingTime() (time.Duration, error) {
// 	// var currentSession MCCCurrentSession
// 	// if err := mcc.getEscapedJSON(mcc.apiURL(mccAPICurrentSession), &currentSession); err != nil {
// 	// 	return 0, err
// 	// }

// 	// return time.Duration(currentSession.Duration * time.Second), nil
// 	return 0, nil
// }

var _ api.MeterCurrent = (*EEBUSCharger)(nil)

// Currents implements the api.MeterCurrent interface
func (eeb *EEBUSCharger) Currents() (float64, float64, float64, error) {
	data, err := eebus.Instance.GetData(eeb.ski)
	if err != nil {
		return 0, 0, 0, err
	}

	return data.EVData.Measurements.CurrentL1, data.EVData.Measurements.CurrentL2, data.EVData.Measurements.CurrentL3, nil
}
