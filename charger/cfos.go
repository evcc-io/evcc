package charger

import (
	"encoding/binary"
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	cfosRegDisconnectCp = 8086
	cfosRegRelaySelect  = 8087
	cfosRegStatus       = 8092
	cfosRegMaxCurrent   = 8093
	cfosRegEnable       = 8094
	cfosRegLastRfid     = 8096
	cfosRegMeter        = 8112
	cfosRegSolarEnabled = 8113

	cfosRegMeterFlags = 8057
	cfosRegEnergy     = 8058 //	4 rw Aktiver Import [Wh]
	cfosRegPower      = 8062 //	2 r	Aktive Leistung [W]
	cfosRegCurrents   = 8064 //	2 r	Momentaner Strom L1 [0.1 A]
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

//go:generate decorate -f decorateCfos -b *CfosPowerBrain -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

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
func NewCfosPowerBrain(uri string, id uint8) (api.Charger, error) {
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

	// decorate meter
	var (
		power    func() (float64, error)
		energy   func() (float64, error)
		currents func() (float64, float64, float64, error)
	)
	if b, err := wb.conn.ReadHoldingRegisters(cfosRegMeter, 1); err == nil && binary.BigEndian.Uint16(b) != 0 {
		power = wb.currentPower
		energy = wb.totalEnergy

		if b, err := wb.conn.ReadHoldingRegisters(cfosRegMeterFlags, 1); err == nil && binary.BigEndian.Uint16(b) != 0 {
			currents = wb.currents
		}
	}

	// decorate phases
	var phases1p3p func(int) error
	if b, err := wb.conn.ReadHoldingRegisters(cfosRegSolarEnabled, 1); err == nil && binary.BigEndian.Uint16(b)&(1<<8) != 0 {
		phases1p3p = wb.phases1p3p
	}

	return decorateCfos(wb, power, energy, currents, phases1p3p), nil
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

// currentPower implements the api.Meter interface
func (wb *CfosPowerBrain) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *CfosPowerBrain) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint64(b)) / 1e3, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *CfosPowerBrain) currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegCurrents, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / 10
	}

	return res[0], res[1], res[2], nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *CfosPowerBrain) phases1p3p(phases int) error {
	if phases == 3 {
		phases = 0
	}
	_, err := wb.conn.WriteSingleRegister(cfosRegRelaySelect, uint16(phases))
	return err
}

var _ api.Resurrector = (*CfosPowerBrain)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *CfosPowerBrain) WakeUp() error {
	_, err := wb.conn.WriteSingleRegister(cfosRegDisconnectCp, uint16(5))
	return err
}

var _ api.Identifier = (*CfosPowerBrain)(nil)

// Identify implements the api.Identifier interface
func (wb *CfosPowerBrain) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegLastRfid, 15)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}
