package charger

import (
	"errors"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/grid-x/modbus"
)

// OpenWB charger implementation
type OpenWB struct {
	log     *util.Logger
	client  modbus.Client
	handler modbus.ClientHandler
}

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// OpenWB is a clone of SimpleEVSE

// TODO generation of setters is not yet functional
// go:generate go run ../cmd/tools/decorate.go -p charger -f decorateOpenWB -b api.Charger -o openwb_decorators -t "api.ChargePhases,Phases1p3p,func(int64) error"

// NewOpenWBFromConfig creates a OpenWB charger from generic config
func NewOpenWBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI, Device string
		Phases      bool `yaml:"1p3p"`
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewOpenWB(cc.URI, cc.Device, cc.Phases)
}

// NewOpenWB creates OpenWB charger
func NewOpenWB(conn, device string, phases bool) (api.Charger, error) {
	log := util.NewLogger("openwb")

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

	owb := &OpenWB{
		log:     log,
		client:  modbus.NewClient(handler),
		handler: handler,
	}

	var phasesS func(int64) error
	if phases {
		phasesS = owb.phases1p3p
	}

	return decorateOpenWB(owb, phasesS), nil
}

// Prepare for bus operation
func (owb *OpenWB) Prepare() {
	if h, ok := owb.handler.(*modbus.TCPClientHandler); ok {
		h.SlaveID = 1
	} else if h, ok := owb.handler.(*modbus.RTUClientHandler); ok {
		h.SlaveID = 1
	}

	time.Sleep(100 * time.Millisecond)
}

// Status implements the Charger.Status interface
func (owb *OpenWB) Status() (api.ChargeStatus, error) {
	owb.Prepare()
	b, err := owb.client.ReadHoldingRegisters(evseRegVehicleStatus, 1)
	owb.log.TRACE.Printf("read status (%d): %0 X", evseRegVehicleStatus, b)
	if err != nil {
		owb.handler.Close()
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
func (owb *OpenWB) Enabled() (bool, error) {
	owb.Prepare()
	b, err := owb.client.ReadHoldingRegisters(evseRegTurnOff, 1)
	owb.log.TRACE.Printf("read charge enable (%d): %0 X", evseRegTurnOff, b)
	if err != nil {
		owb.handler.Close()
		return false, err
	}

	return b[1] == 1, nil
}

// Enable implements the Charger.Enable interface
func (owb *OpenWB) Enable(enable bool) error {
	owb.Prepare()
	b, err := owb.client.ReadHoldingRegisters(evseRegTurnOff, 1)
	owb.log.TRACE.Printf("read charge enable (%d): %0 X", evseRegTurnOff, b)
	if err != nil {
		owb.handler.Close()
		return err
	}

	if enable {
		b[1] |= 1
	} else {
		b[1] &= ^byte(1)
	}

	b, err = owb.client.WriteMultipleRegisters(evseRegTurnOff, 1, b)
	owb.log.TRACE.Printf("write charge enable %d %0X: %0 X", evseRegTurnOff, b, b)
	if err != nil {
		owb.handler.Close()
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (owb *OpenWB) MaxCurrent(current int64) error {
	b := []byte{0, byte(current)}

	owb.Prepare()
	b, err := owb.client.WriteMultipleRegisters(evseRegAmpsConfig, 1, b)
	owb.log.TRACE.Printf("write max current %d %0X: %0 X", evseRegAmpsConfig, current, b)
	if err != nil {
		owb.handler.Close()
	}

	return err
}

// Phases1p3p implements the Charger.Phases1p3p interface
func (owb *OpenWB) phases1p3p(int64) error {
	return errors.New("not implemented")
}
