package charger

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	cfosRegStatus     = 8092 // Holding
	cfosRegMaxCurrent = 8093 // Holding
	cfosRegEnable     = 8094 // Holding
)

// CfosPowerBrain is an charger implementation for cFos PowerBrain wallboxes.
// It uses Modbus TCP to communicate at modbus client id 1 and power meters at id 2 and 3.
// https://www.cfos-emobility.de/en-gb/cfos-power-brain/modbus-registers.htm
type CfosPowerBrain struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("cfos", NewCfosPowerBrainFromConfig)
}

// NewCfosPowerBrainFromConfig creates a cFos charger from generic config
func NewCfosPowerBrainFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewCfosPowerBrain(cc.URI, cc.ID)
}

// NewCfosPowerBrain creates a cFos charger
func NewCfosPowerBrain(uri string, id uint8) (*CfosPowerBrain, error) {
	uri = util.DefaultPort(uri, 4701)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("cfos")
	conn.Logger(log.TRACE)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	wb := &CfosPowerBrain{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *CfosPowerBrain) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch b[1] {
	case 0: // warten
		return api.StatusA, nil
	case 1: // Fahrzeug erkannt
		return api.StatusB, nil
	case 2: // laden
		return api.StatusC, nil
	case 3: // laden mit Kühlung
		return api.StatusD, nil
	case 4: // kein Strom
		return api.StatusE, nil
	case 5: // Fehler
		return api.StatusF, nil
	default:
		return api.StatusNone, errors.New("invalid response")
	}
}

// Enabled implements the api.Charger interface
func (wb *CfosPowerBrain) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[1] == 1, nil
}

// Enable implements the api.Charger interface
func (wb *CfosPowerBrain) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(cfosRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *CfosPowerBrain) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*CfosPowerBrain)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *CfosPowerBrain) MaxCurrentMillis(current float64) error {
	_, err := wb.conn.WriteSingleRegister(cfosRegMaxCurrent, uint16(current*10))
	return err
}
