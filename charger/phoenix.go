package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/grid-x/modbus"
)

const (
	phRegStatus     = 100 // Input
	phRegChargeTime = 102 // Input
	phRegMaxCurrent = 300 // Holding
	phRegEnable     = 400 // Coil
)

// Phoenix is an api.ChargeController implementation for Phoenix wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Phoenix struct {
	log     *api.Logger
	client  modbus.Client
	handler *modbus.TCPClientHandler
}

// NewPhoenixFromConfig creates a Phoenix charger from generic config
func NewPhoenixFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ URI string }{}
	decodeOther(log, other, &cc)

	return NewPhoenix(cc.URI)
}

// NewPhoenix creates a Phoenix charger
func NewPhoenix(conn string) api.Charger {
	log := api.NewLogger("phoe")
	if conn == "" {
		log.FATAL.Fatal("missing connection")
	}

	handler := modbus.NewTCPClientHandler(conn)
	client := modbus.NewClient(handler)

	handler.SlaveID = slaveID
	handler.Timeout = timeout
	handler.ProtocolRecoveryTimeout = protocolTimeout

	wb := &Phoenix{
		log:     log,
		client:  client,
		handler: handler,
	}

	return wb
}

// Status implements the Charger.Status interface
func (wb *Phoenix) Status() (api.ChargeStatus, error) {
	b, err := wb.client.ReadInputRegisters(phRegStatus, 1)
	wb.log.TRACE.Printf("read status (%d): %0 X", phRegStatus, b)
	if err != nil {
		wb.handler.Close()
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *Phoenix) Enabled() (bool, error) {
	b, err := wb.client.ReadCoils(phRegEnable, 1)
	wb.log.TRACE.Printf("read charge enable (%d): %0 X", phRegEnable, b)
	if err != nil {
		wb.handler.Close()
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *Phoenix) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	b, err := wb.client.WriteSingleCoil(phRegEnable, u)
	wb.log.TRACE.Printf("write charge enable %d %0X: %0 X", phRegEnable, u, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *Phoenix) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	b, err := wb.client.WriteSingleRegister(phRegMaxCurrent, uint16(current))
	wb.log.TRACE.Printf("write max current %d %0X: %0 X", phRegMaxCurrent, current, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// ChargingTime yields current charge run duration
func (wb *Phoenix) ChargingTime() (time.Duration, error) {
	b, err := wb.client.ReadInputRegisters(phRegChargeTime, 2)
	wb.log.TRACE.Printf("read charge time (%d): %0 X", phRegChargeTime, b)
	if err != nil {
		wb.handler.Close()
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}
