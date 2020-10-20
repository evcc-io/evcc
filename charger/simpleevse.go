package charger

import (
	"errors"
	"time"

	"github.com/mark-sch/evcc/api"
	"github.com/mark-sch/evcc/util"
	"github.com/mark-sch/evcc/util/modbus"
)

// SimpleEVSE charger implementation
type SimpleEVSE struct {
	log     *util.Logger
	conn    *modbus.Connection
	current int64
}

const (
	evseRegAmpsConfig    = 1000
	evseRegVehicleStatus = 1002
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

	conn.Logger(log.TRACE)
	conn.Delay(200 * time.Millisecond)

	evse := &SimpleEVSE{
		log:     log,
		conn:    conn,
		current: 6, // assume min current
	}

	return evse, nil
}

// Status implements the Charger.Status interface
func (evse *SimpleEVSE) Status() (api.ChargeStatus, error) {
	b, err := evse.conn.ReadHoldingRegisters(evseRegVehicleStatus, 1)
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
	b, err := evse.conn.ReadHoldingRegisters(evseRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	enabled := b[1] != 0
	if enabled {
		evse.current = int64(b[1])
	}

	return enabled, nil
}

// Enable implements the Charger.Enable interface
func (evse *SimpleEVSE) Enable(enable bool) error {
	b := []byte{0, 0}

	if enable {
		b[1] = byte(evse.current)
	}

	_, err := evse.conn.WriteMultipleRegisters(evseRegAmpsConfig, 1, b)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (evse *SimpleEVSE) MaxCurrent(current int64) error {
	b := []byte{0, byte(current)}

	_, err := evse.conn.WriteMultipleRegisters(evseRegAmpsConfig, 1, b)
	if err == nil {
		evse.current = current
	}

	return err
}
