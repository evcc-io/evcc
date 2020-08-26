package charger

import (
	"errors"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/grid-x/modbus"
)

// SimpleEVSE charger implementation
type SimpleEVSE struct {
	log     *util.Logger
	client  modbus.Client
	handler modbus.ClientHandler
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
	cc := struct{ URI, Device string }{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSimpleEVSE(cc.URI, cc.Device)
}

// NewSimpleEVSE creates SimpleEVSE charger
func NewSimpleEVSE(conn, device string) (api.Charger, error) {
	log := util.NewLogger("evse")

	var handler modbus.ClientHandler
	if conn != "" && device != "" {
		return nil, errors.New("cannot define uri and device both")
	}
	if conn != "" {
		handler = modbus.NewTCPClientHandler(conn)
		handler.(*modbus.TCPClientHandler).Timeout = time.Second
	}
	if device != "" {
		handler = modbus.NewRTUClientHandler(device)
		handler.(*modbus.RTUClientHandler).BaudRate = 9600
		handler.(*modbus.RTUClientHandler).DataBits = 8
		handler.(*modbus.RTUClientHandler).StopBits = 1
		handler.(*modbus.RTUClientHandler).Parity = "N"
		handler.(*modbus.RTUClientHandler).Timeout = time.Second
	}
	if handler == nil {
		return nil, errors.New("must define either uri or device")
	}

	evse := &SimpleEVSE{
		log:     log,
		client:  modbus.NewClient(handler),
		handler: handler,
	}

	return evse, nil
}

// Prepare for bus operation
func (evse *SimpleEVSE) Prepare() {
	if h, ok := evse.handler.(*modbus.TCPClientHandler); ok {
		h.SlaveID = 1
	} else if h, ok := evse.handler.(*modbus.RTUClientHandler); ok {
		h.SlaveID = 1
	}

	time.Sleep(100 * time.Millisecond)
}

// Status implements the Charger.Status interface
func (evse *SimpleEVSE) Status() (api.ChargeStatus, error) {
	evse.Prepare()
	b, err := evse.client.ReadHoldingRegisters(evseRegVehicleStatus, 1)
	evse.log.TRACE.Printf("read status (%d): %0 X", evseRegVehicleStatus, b)
	if err != nil {
		evse.handler.Close()
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
	evse.Prepare()
	b, err := evse.client.ReadHoldingRegisters(evseRegTurnOff, 1)
	evse.log.TRACE.Printf("read charge enable (%d): %0 X", evseRegTurnOff, b)
	if err != nil {
		evse.handler.Close()
		return false, err
	}

	return b[1] == 1, nil
}

// Enable implements the Charger.Enable interface
func (evse *SimpleEVSE) Enable(enable bool) error {
	evse.Prepare()
	b, err := evse.client.ReadHoldingRegisters(evseRegTurnOff, 1)
	evse.log.TRACE.Printf("read charge enable (%d): %0 X", evseRegTurnOff, b)
	if err != nil {
		evse.handler.Close()
		return err
	}

	if enable {
		b[1] |= 1
	} else {
		b[1] &= ^byte(1)
	}

	b, err = evse.client.WriteMultipleRegisters(evseRegTurnOff, 1, b)
	evse.log.TRACE.Printf("write charge enable %d %0X: %0 X", evseRegTurnOff, b, b)
	if err != nil {
		evse.handler.Close()
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (evse *SimpleEVSE) MaxCurrent(current int64) error {
	b := []byte{0, byte(current)}

	evse.Prepare()
	b, err := evse.client.WriteMultipleRegisters(evseRegAmpsConfig, 1, b)
	evse.log.TRACE.Printf("write max current %d %0X: %0 X", evseRegAmpsConfig, current, b)
	if err != nil {
		evse.handler.Close()
	}

	return err
}
