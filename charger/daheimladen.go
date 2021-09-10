package charger

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// DaheimLaden charger implementation
type DaheimLaden struct {
	*request.Helper
	token string
}

func init() {
	registry.Add("daheimladen", NewDaheimLadenFromConfig)
}

// NewDaheimLadenFromConfig creates a DaheimLaden charger from generic config
func NewDaheimLadenFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Token string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDaheimLaden(cc.Token)
}

// NewDaheimLaden creates DaheimLaden charger
func NewDaheimLaden(token string) (*DaheimLaden, error) {
	c := &DaheimLaden{
		Helper: request.NewHelper(util.NewLogger("daheim")),
		token:  token,
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *DaheimLaden) Enabled() (bool, error) {
	return false, errors.New("not implemented")
}

// Enable implements the api.Charger interface
func (c *DaheimLaden) Enable(enable bool) error {
	return errors.New("not implemented")
}

// MaxCurrent implements the api.Charger interface
func (c *DaheimLaden) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current) * 1e3)
}

var _ api.ChargerEx = (*DaheimLaden)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *DaheimLaden) MaxCurrentMillis(current float64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *DaheimLaden) Status() (api.ChargeStatus, error) {
	return api.StatusB, errors.New("not implemented")
}

var _ api.Meter = (*DaheimLaden)(nil)

// CurrentPower implements the api.Meter interface
func (c *DaheimLaden) CurrentPower() (float64, error) {
	return 0, errors.New("not implemented")
}

var _ api.MeterEnergy = (*DaheimLaden)(nil)

// TotalEnergy implements the api.MeterMeterEnergy interface
func (c *DaheimLaden) TotalEnergy() (float64, error) {
	return 0, errors.New("not implemented")
}

var _ api.MeterCurrent = (*DaheimLaden)(nil)

// Currents implements the api.MeterCurrent interface
func (c *DaheimLaden) Currents() (float64, float64, float64, error) {
	return 0, 0, 0, errors.New("not implemented")
}

var _ api.Identifier = (*DaheimLaden)(nil)

// Identify implements the api.Identifier interface
func (c *DaheimLaden) Identify() (string, error) {
	return "", errors.New("not implemented")
}
