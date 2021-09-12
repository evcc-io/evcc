package charger

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
)

// OCPP charger implementation
type OCPP struct {
	log *util.Logger
	cp  *ocpp.CP
}

func init() {
	registry.Add("ocpp", NewOCPPFromConfig)
}

// NewOCPPFromConfig creates a OCPP charger from generic config
func NewOCPPFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		ChargePoint string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOCPP(cc.ChargePoint)
}

// NewOCPP creates OCPP charger
func NewOCPP(cp string) (*OCPP, error) {
	c := &OCPP{
		log: util.NewLogger("ocpp-" + cp),
		cp:  ocpp.Instance().Register(cp),
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *OCPP) Enabled() (bool, error) {
	return false, errors.New("not implemented")
}

// Enable implements the api.Charger interface
func (c *OCPP) Enable(enable bool) error {
	return errors.New("not implemented")
}

// MaxCurrent implements the api.Charger interface
func (c *OCPP) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current) * 1e3)
}

var _ api.ChargerEx = (*OCPP)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *OCPP) MaxCurrentMillis(current float64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *OCPP) Status() (api.ChargeStatus, error) {
	return api.StatusB, errors.New("not implemented")
}

var _ api.Meter = (*OCPP)(nil)

// CurrentPower implements the api.Meter interface
func (c *OCPP) CurrentPower() (float64, error) {
	return 0, errors.New("not implemented")
}

var _ api.MeterEnergy = (*OCPP)(nil)

// TotalEnergy implements the api.MeterMeterEnergy interface
func (c *OCPP) TotalEnergy() (float64, error) {
	return 0, errors.New("not implemented")
}

var _ api.MeterCurrent = (*OCPP)(nil)

// Currents implements the api.MeterCurrent interface
func (c *OCPP) Currents() (float64, float64, float64, error) {
	return 0, 0, 0, errors.New("not implemented")
}

var _ api.Identifier = (*OCPP)(nil)

// Identify implements the api.Identifier interface
func (c *OCPP) Identify() (string, error) {
	return "", errors.New("not implemented")
}
