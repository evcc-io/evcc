// +build eebus

package charger

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/amp-x/eebus/communication"
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/charger/eebus"
	"github.com/andig/evcc/util"
	// "github.com/andig/evcc/util/sponsor"
)

type EEBus struct {
	log        *util.Logger
	cc         *communication.ConnectionController
	maxCurrent float64
}

func init() {
	registry.Add("eebus", NewEEBusFromConfig)
}

// NewEEBusFromConfig creates an EEBus charger from generic config
func NewEEBusFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Ski         string
		Certificate struct {
			Public, Private []byte
		}
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(cc.Certificate.Public, cc.Certificate.Private)
	if err != nil {
		return nil, err
	}

	return NewEEBus(cc.Ski, cert)
}

// NewMobileConnect creates MCC charger
func NewEEBus(ski string, cert tls.Certificate) (*EEBus, error) {
	log := util.NewLogger("eebus")

	// if !sponsor.IsAuthorized() {
	// 	return nil, errors.New("eebus requires evcc sponsorship, register at https://cloud.evcc.io")
	// }

	if eebus.Instance == nil {
		var err error
		if eebus.Instance, err = eebus.New(log, cert); err != nil {
			return nil, err
		}

		go eebus.Instance.Run()
	}

	c := &EEBus{log: log}

	eebus.Instance.Register(ski, c.onConnect)

	// on start we need to disable charging as it would otherwise start with max current
	_ = c.Enable(false)

	return c, nil
}

func (c *EEBus) onConnect(cc *communication.ConnectionController) {
	c.cc = cc
}

// Status implements the api.Charger interface
func (c *EEBus) Status() (api.ChargeStatus, error) {
	data, err := c.cc.GetData()
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
func (c *EEBus) Enabled() (bool, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return false, err
	}

	if data.EVData.ChargeState == communication.EVChargeStateEnumTypeUnplugged {
		return false, nil
	}

	// when stopping charging by sending default current values to L1, it looks like the
	// Taycan OBC sets current to 0.5 and power varies between 1-3W
	// on enabling with 2A on L1, the measurement e..g goes:
	//   18:19:57 set limit on L1 to 2A
	//   18:19:57 measurement on L1 1.2A - 170W
	//   18:19:59 measurement on L1 0.5A - 3W
	//   18:20:01 measurement on L1 0.5A - 3W
	//   18:20:02 measurement on L1 2A - 450W
	// so it took 5 seconds to reach the low setting. if we check enabled in between, it may appear as disabled!
	if data.EVData.Measurements.CurrentL1 > 0.5 ||
		data.EVData.Measurements.CurrentL2 > 0.5 ||
		data.EVData.Measurements.CurrentL3 > 0.5 {
		return true, nil
	}

	return false, nil
}

// Enable implements the api.Charger interface
func (c *EEBus) Enable(enable bool) error {
	data, err := c.cc.GetData()
	if err != nil {
		return err
	}

	if !enable {
		// Important notes on enabling/disabling!!
		// ISO15118 mode:
		//   non-asymmetric or all phases set to 0: the OBC will wait for 1 minute, if the values remain after 1 min, it will pause then
		//   asymmetric and only some phases set to 0: no pauses or waiting for changes required
		//   asymmetric mode requires Plug & Charge (PnC) and Value Added Services (VAS)
		// IEC61851 mode:
		//   switching between 1/3 phases: stop charging, pause for 2 minutes, change phases, resume charging
		//   frequent switching should be avoided by all means!
		c.maxCurrent = 0
		c.cc.WriteCurrentLimitData([]float64{0.0, 0.0, 0.0}, data.EVData)
	}

	// if we set MaxCurrent > Min value and then try to enable the charger, it would reset it to min
	if c.maxCurrent > 0 {
		c.cc.WriteCurrentLimitData([]float64{c.maxCurrent, c.maxCurrent, c.maxCurrent}, data.EVData)
	} else {
		c.cc.WriteCurrentLimitData([]float64{data.EVData.LimitsL1.Min, data.EVData.LimitsL2.Min, data.EVData.LimitsL3.Min}, data.EVData)
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *EEBus) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EEBus)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *EEBus) MaxCurrentMillis(current float64) error {
	data, err := c.cc.GetData()
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

	c.maxCurrent = current

	// TODO error handling
	c.cc.WriteCurrentLimitData([]float64{current, current, current}, data.EVData)

	return nil
}

var _ api.Meter = (*EEBus)(nil)

// CurrentPower implements the api.Meter interface
func (c *EEBus) CurrentPower() (float64, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return 0, err
	}

	power := data.EVData.Measurements.PowerL1 + data.EVData.Measurements.PowerL2 + data.EVData.Measurements.PowerL3

	return power, nil
}

var _ api.ChargeRater = (*EEBus)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *EEBus) ChargedEnergy() (float64, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return 0, err
	}

	return data.EVData.Measurements.ChargedEnergy, nil
}

// var _ api.ChargeTimer = (*EEBus)(nil)

// // ChargingTime implements the api.ChargeTimer interface
// func (c *EEBus) ChargingTime() (time.Duration, error) {
// 	// var currentSession MCCCurrentSession
// 	// if err := mcc.getEscapedJSON(mcc.apiURL(mccAPICurrentSession), &currentSession); err != nil {
// 	// 	return 0, err
// 	// }

// 	// return time.Duration(currentSession.Duration * time.Second), nil
// 	return 0, nil
// }

var _ api.MeterCurrent = (*EEBus)(nil)

// Currents implements the api.MeterCurrent interface
func (c *EEBus) Currents() (float64, float64, float64, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return 0, 0, 0, err
	}

	return data.EVData.Measurements.CurrentL1, data.EVData.Measurements.CurrentL2, data.EVData.Measurements.CurrentL3, nil
}

var _ api.Identifier = (*EEBus)(nil)

// Identifier implements the api.Identifier interface
func (c *EEBus) Identify() (string, error) {
	data, err := c.cc.GetData()
	if err != nil {
		return "", err
	}

	if len(data.EVData.Identification) > 0 {
		return data.EVData.Identification, nil
	}

	return "", nil
}
