package charger

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	dadapowerRegFailsafeTimeout     = 102
	dadapowerRegModel               = 105
	dadapowerRegSerial              = 106 // 6
	dadapowerRegFirmware            = 112 // 6
	dadapowerRegChargingAllowed     = 1000
	dadapowerRegChargeCurrentLimit  = 1001
	dadapowerRegActivePhases        = 1002
	dadapowerRegCurrents            = 1006
	dadapowerRegActiveEnergy        = 1009
	dadapowerRegPlugState           = 1016
	dadapowerRegEnergyImportSession = 1017
	dadapowerRegEnergyImportTotal   = 1025
	dadapowerRegIdentification      = 1040 // 20
)

// Dadapower charger implementation
type Dadapower struct {
	log       *util.Logger
	conn      *modbus.Connection
	regOffset uint16
}

func init() {
	registry.Add("dadapower", NewDadapowerFromConfig)
}

// NewDadapowerFromConfig creates a Dadapower charger from generic config
func NewDadapowerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDadapower(cc.URI, cc.ID)
}

// NewDadapower creates a Dadapower charger
func NewDadapower(uri string, id uint8) (*Dadapower, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("dadapower")
	conn.Logger(log.TRACE)

	wb := &Dadapower{
		log:  log,
		conn: conn,
	}

	// 5min failsafe timeout
	if _, err := wb.conn.WriteSingleRegister(dadapowerRegFailsafeTimeout, 5*60); err != nil {
		return nil, fmt.Errorf("could not set failsafe timeout: %v", err)
	}

	// The charging station may have multiple charging ports - use offset for register addresses for each port
	if id > 1 {
		wb.regOffset = (uint16(id) - 1) * 1000
	}

	go wb.heartbeat()

	return wb, nil
}

func (wb *Dadapower) heartbeat() {
	for range time.Tick(time.Minute) {
		if _, err := wb.conn.ReadInputRegisters(dadapowerRegFailsafeTimeout, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Dadapower) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(dadapowerRegPlugState+wb.regOffset, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch binary.BigEndian.Uint16(b) {
	case 0x0A: // ready
		return api.StatusA, nil
	case 0x0B: // EV is present
		return api.StatusB, nil
	case 0x0C: // charging
		return api.StatusC, nil
	case 0x0D: // charging with ventilation
		return api.StatusD, nil
	case 0x0E: // failure (e.g. diode check, RCD failure)
		return api.StatusE, nil
	default:
		return api.StatusNone, errors.New("invalid response")
	}
}

// Enabled implements the api.Charger interface
func (wb *Dadapower) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(dadapowerRegChargingAllowed+wb.regOffset, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Dadapower) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(dadapowerRegChargingAllowed+wb.regOffset, u)

	return err
}

var _ api.ChargerEx = (*Dadapower)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Dadapower) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %v", current)
	}

	_, err := wb.conn.WriteSingleRegister(dadapowerRegChargeCurrentLimit+wb.regOffset, uint16(current*100))

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Dadapower) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.Meter = (*Dadapower)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Dadapower) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(dadapowerRegActiveEnergy+wb.regOffset, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*Dadapower)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Dadapower) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(dadapowerRegEnergyImportTotal+wb.regOffset, 4)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint64(b)) / 1e3, err
}

var _ api.ChargeRater = (*Dadapower)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Dadapower) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(dadapowerRegEnergyImportSession+wb.regOffset, 4)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint64(b)) / 1e3, err
}

var _ api.PhaseCurrents = (*Dadapower)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Dadapower) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(dadapowerRegCurrents+wb.regOffset, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(binary.BigEndian.Uint16(b)) / 100
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseSwitcher = (*Dadapower)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Dadapower) Phases1p3p(phases int) error {
	enabled, err := wb.Enabled()
	if err != nil {
		return err
	}

	if enabled {
		if err := wb.Enable(false); err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}

	if _, err := wb.conn.WriteSingleRegister(dadapowerRegActivePhases+wb.regOffset, uint16(phases)); err != nil {
		return err
	}

	if enabled {
		time.Sleep(2 * time.Second)
		if err := wb.Enable(true); err != nil {
			return err
		}
	}

	return nil
}

var _ api.Identifier = (*Dadapower)(nil)

// Identify implements the api.Identifier interface
func (wb *Dadapower) Identify() (string, error) {
	u, err := wb.conn.ReadInputRegisters(dadapowerRegIdentification+wb.regOffset, 20)
	if err != nil {
		return "", err
	}

	return bytesAsString(u), nil
}

var _ api.Diagnosis = (*Dadapower)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Dadapower) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(dadapowerRegModel, 1); err == nil {
		fmt.Printf("Model:\t%d\n", b[1])
	}
	if b, err := wb.conn.ReadInputRegisters(dadapowerRegSerial, 6); err == nil {
		fmt.Printf("Serial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(dadapowerRegFirmware, 6); err == nil {
		fmt.Printf("Firmware:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(dadapowerRegIdentification, 20); err == nil {
		fmt.Printf("Identification:\t%s\n", b)
	}
}
