package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/grid-x/modbus"
)

const (
	phEVCCRegStatus     = 24000 // Input
	phEVCCRegMaxCurrent = 22000 // Holding
	phEVCCRegEnable     = 20000 // Coil
)

// PhoenixEVCC is an api.ChargeController implementation for Phoenix EV-CC-AC1-M wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type PhoenixEVCC struct {
	log     *util.Logger
	client  modbus.Client
	handler *modbus.TCPClientHandler
}

// NewPhoenixEVCCFromConfig creates a Phoenix charger from generic config
func NewPhoenixEVCCFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	cc := struct {
		Baudrate int
		ID       uint8
	}{}
	util.DecodeOther(log, other, &cc)

	if cc.ID == 0 {
		log.FATAL.Fatal("config: missing slave id")
	}

	return NewPhoenixEVCC(cc.ID, cc.Baudrate)
}

// NewPhoenixEVCC creates a Phoenix charger
func NewPhoenixEVCC(slaveID uint8, baudrate int) api.Charger {
	log := util.NewLogger("evcc")

	handler := modbus.NewTCPClientHandler(conn)
	client := modbus.NewClient(handler)

	handler.SlaveID = slaveID
	handler.Timeout = timeout
	handler.ProtocolRecoveryTimeout = protocolTimeout

	wb := &PhoenixEVCC{
		log:     log,
		client:  client,
		handler: handler,
	}

	return wb
}

// Status implements the Charger.Status interface
func (wb *PhoenixEVCC) Status() (api.ChargeStatus, error) {
	b, err := wb.client.ReadInputRegisters(phEVCCRegStatus, 1)
	wb.log.TRACE.Printf("read status (%d): %0 X", phEVCCRegStatus, b)
	if err != nil {
		wb.handler.Close()
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixEVCC) Enabled() (bool, error) {
	b, err := wb.client.ReadCoils(phEVCCRegEnable, 1)
	wb.log.TRACE.Printf("read charge enable (%d): %0 X", phEVCCRegEnable, b)
	if err != nil {
		wb.handler.Close()
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *PhoenixEVCC) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	b, err := wb.client.WriteSingleCoil(phEVCCRegEnable, u)
	wb.log.TRACE.Printf("write charge enable %d %0X: %0 X", phEVCCRegEnable, u, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixEVCC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	b, err := wb.client.WriteSingleRegister(phEVCCRegMaxCurrent, uint16(current))
	wb.log.TRACE.Printf("write max current %d %0X: %0 X", phEVCCRegMaxCurrent, current, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}
