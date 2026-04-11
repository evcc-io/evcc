package charger

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/fritzdect"
	"github.com/evcc-io/evcc/util"
)

// FRITZ! FritzBox AHA interface specifications:
// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf
// https://fritz.support/resources/SmarthomeRestApiFRITZOS82.html (REST API for FritzOS 8.2+)

// FritzDECT charger implementation
type FritzDECT struct {
	conn fritzdect.SwitchAPI
	*switchSocket
}

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect charger from generic config
func NewFritzDECTFromConfig(other map[string]any) (api.Charger, error) {
	var cc struct {
		embed              `mapstructure:",squash"`
		fritzdect.Settings `mapstructure:",squash"`
		StandbyPower       float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewFritzDECT(cc.embed, cc.URI, cc.AIN, cc.User, cc.Password, cc.StandbyPower, cc.Legacy)
}

// NewFritzDECT creates a new connection with standbypower for charger
func NewFritzDECT(embed embed, uri, ain, user, password string, standbypower float64, legacy bool) (*FritzDECT, error) {
	var conn fritzdect.SwitchAPI
	var err error

	// Use legacy LUA API if explicitly requested, otherwise use new REST API
	if legacy {
		conn, err = fritzdect.NewConnection(uri, ain, user, password)
	} else {
		conn, err = fritzdect.NewRestConnection(uri, ain, user, password)
	}
	if err != nil {
		return nil, err
	}

	c := &FritzDECT{
		conn: conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, nil
}

// Status implements the api.Charger interface
func (c *FritzDECT) Status() (api.ChargeStatus, error) {
	present, err := c.conn.SwitchPresent()
	if err != nil {
		return api.StatusNone, err
	}

	if !present {
		return api.StatusNone, api.ErrNotAvailable
	}

	return c.switchSocket.Status()
}

// Enabled implements the api.Charger interface
func (c *FritzDECT) Enabled() (bool, error) {
	return c.conn.SwitchState()
}

// Enable implements the api.Charger interface
func (c *FritzDECT) Enable(enable bool) error {
	if enable {
		return c.conn.SwitchOn()
	}
	return c.conn.SwitchOff()
}

var _ api.MeterEnergy = (*FritzDECT)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *FritzDECT) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
