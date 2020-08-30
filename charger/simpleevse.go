package charger

import (
	"errors"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

// SimpleEVSE charger implementation
type SimpleEVSE struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	evseRegAmpsConfig    = 1000
	evseRegVehicleStatus = 1002
	evseRegTurnOff       = 1004
)

func init() {
	registry.Add("simpleevse", NewSimpleEVSEFromConfig)
}

// https://files.ev-power.eu/inc/_doc/attach/StoItem/4418/evse-wb-din_Manual.pdf

// NewSimpleEVSEFromConfig creates a SimpleEVSE charger from generic config
func NewSimpleEVSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		Baudrate: 9600,
		Comset:   "8N1",
		ID:       1,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSimpleEVSE(cc.URI, cc.Device, cc.Comset, cc.Baudrate, true, cc.ID)
}

// NewSimpleEVSE creates SimpleEVSE charger
func NewSimpleEVSE(uri, device, comset string, baudrate int, rtu bool, slaveID uint8) (api.Charger, error) {
	log := util.NewLogger("evse")

	conn, err := modbus.NewConnection(uri, device, comset, baudrate, rtu, slaveID)
	if err != nil {
		return nil, err
	}

	evse := &SimpleEVSE{
		log:  log,
		conn: conn,
	}

	return evse, nil
}

// Status implements the Charger.Status interface
func (evse *SimpleEVSE) Status() (api.ChargeStatus, error) {
	b, err := evse.conn.ReadHoldingRegisters(evseRegVehicleStatus, 1)
	evse.log.TRACE.Printf("read status (%d): %0 X", evseRegVehicleStatus, b)
	if err != nil {
		return api.StatusNone, err
	}

	switch b[1] {
	case 1: // ready
		return api.StatusA, nil
	case 2: // EV is present
		return api.StatusB, nil
	case 3: // charging
		return api.StatusC, nil
	case 4: // charging with ventilation
		return api.StatusD, nil
	case 5: // failure (e.g. diode check, RCD failure)
		return api.StatusE, nil
	default:
		return api.StatusNone, errors.New("invalid response")
	}
}

// Enabled implements the Charger.Enabled interface
func (evse *SimpleEVSE) Enabled() (bool, error) {
	b, err := evse.conn.ReadHoldingRegisters(evseRegTurnOff, 1)
	evse.log.TRACE.Printf("read charge enable (%d): %0 X", evseRegTurnOff, b)
	if err != nil {
		return false, err
	}

	return b[1] == 1, nil
}

// Enable implements the Charger.Enable interface
func (evse *SimpleEVSE) Enable(enable bool) error {
	b, err := evse.conn.ReadHoldingRegisters(evseRegTurnOff, 1)
	evse.log.TRACE.Printf("read charge enable (%d): %0 X", evseRegTurnOff, b)
	if err != nil {
		return err
	}

	if enable {
		b[1] |= 1
	} else {
		b[1] &= ^byte(1)
	}

	bb, err := evse.conn.WriteMultipleRegisters(evseRegTurnOff, 1, b)
	evse.log.TRACE.Printf("write charge enable (%d) %0X: %0 X", evseRegTurnOff, b, bb)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (evse *SimpleEVSE) MaxCurrent(current int64) error {
	b := []byte{0, byte(current)}

	b, err := evse.conn.WriteMultipleRegisters(evseRegAmpsConfig, 1, b)
	evse.log.TRACE.Printf("write max current (%d) %0X: %0 X", evseRegAmpsConfig, current, b)

	return err
}
