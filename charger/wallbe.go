package charger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/grid-x/modbus"
)

const (
	slaveID = 255

	wbRegStatus            = 100 // Input
	wbRegChargeTime        = 102 // Input
	wbRegActualCurrent     = 300 // Holding
	wbRegEnable            = 400 // Coil
	wbRegOverchargeProtect = 409 // Coil
	wbRegReset             = 413 // Coil
	wbRegMaxCurrent        = 528 // Holding

	timeout         = 1 * time.Second
	protocolTimeout = 2 * time.Second
)

// Wallbe is an api.ChargeController implementation for Wallbe wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Wallbe struct {
	log     *api.Logger
	client  modbus.Client
	handler *modbus.TCPClientHandler
}

// NewWallbeFromConfig creates a Wallbe charger from generic config
func NewWallbeFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ URI string }{}
	api.DecodeOther(log, other, &cc)

	return NewWallbe(cc.URI)
}

// NewWallbe creates a Wallbe charger
func NewWallbe(conn string) api.Charger {
	if conn == "" {
		conn = "192.168.0.8:502"
	}

	handler := modbus.NewTCPClientHandler(conn)
	client := modbus.NewClient(handler)

	handler.SlaveID = slaveID
	handler.Timeout = timeout
	handler.ProtocolRecoveryTimeout = protocolTimeout

	wb := &Wallbe{
		log:     api.NewLogger("wlbe"),
		client:  client,
		handler: handler,
	}

	// wb.showIOs()

	return wb
}

// showIOs logs all input/output register values and their configurations
func (wb *Wallbe) showIOs() {
	// inputs
	wb.showIO("LD", 520, 200) // 200 = EN?
	wb.showIO("EN", 521, 201) // 201 = XR?
	wb.showIO("ML", 522, 202) // 202 = LD?
	wb.showIO("XR", 523, 203) // 203 = ML?
	wb.showIO("IN", 524, 208)

	// outputs
	wb.showIO("ER", 327, 204)
	wb.showIO("LR", 328, 205)
	wb.showIO("VR", 329, 206)
	wb.showIO("CR", 330, 207)

	if b, err := wb.client.ReadHoldingRegisters(390, 1); err != nil {
		wb.log.FATAL.Printf("%s definition %v", "Schütz", err)
	} else {
		wb.log.DEBUG.Printf("%s (%d)", "Schütz", binary.BigEndian.Uint16(b))
	}
}

// showIOs logs a single input/output register's values and their configurations
func (wb *Wallbe) showIO(input string, definition uint16, status uint16) {
	var def uint16
	var val byte

	b, err := wb.client.ReadHoldingRegisters(definition, 1)
	if err != nil {
		wb.log.FATAL.Printf("%s definition %v", input, err)
		return
	}
	def = binary.BigEndian.Uint16(b)

	b, err = wb.client.ReadDiscreteInputs(status, 1)
	if err != nil {
		wb.log.FATAL.Printf("%s status %v", input, err)
		return
	}
	val = b[0]

	wb.log.DEBUG.Printf("%s = %d (%d)", input, val, def)
}

// Status implements the Charger.Status interface
func (wb *Wallbe) Status() (api.ChargeStatus, error) {
	b, err := wb.client.ReadInputRegisters(wbRegStatus, 1)
	wb.log.TRACE.Printf("read status (%d): %0 X", wbRegStatus, b)
	if err != nil {
		wb.handler.Close()
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *Wallbe) Enabled() (bool, error) {
	b, err := wb.client.ReadCoils(wbRegEnable, 1)
	wb.log.TRACE.Printf("read charge enable (%d): %0 X", wbRegEnable, b)
	if err != nil {
		wb.handler.Close()
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *Wallbe) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	b, err := wb.client.WriteSingleCoil(wbRegEnable, u)
	wb.log.TRACE.Printf("write charge enable %d %0X: %0 X", wbRegEnable, u, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *Wallbe) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current * 10)

	b, err := wb.client.WriteSingleRegister(wbRegMaxCurrent, u)
	wb.log.TRACE.Printf("write max current %d %0X: %0 X", wbRegMaxCurrent, u, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// ChargingTime yields current charge run duration
func (wb *Wallbe) ChargingTime() (time.Duration, error) {
	b, err := wb.client.ReadInputRegisters(wbRegChargeTime, 2)
	wb.log.TRACE.Printf("read charge time (%d): %0 X", wbRegChargeTime, b)
	if err != nil {
		wb.handler.Close()
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}
