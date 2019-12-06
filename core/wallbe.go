package core

import (
	"encoding/binary"
	"time"

	"github.com/andig/evcc/api"
	"github.com/grid-x/modbus"
)

const (
	slaveID = 255

	regStatus            = 100
	regChargeTime        = 102
	regActualCurrent     = 300
	regOverchargeProtect = 409
	regMaxCurrent        = 528

	timeout         = 1 * time.Second
	protocolTimeout = 2 * time.Second
)

// Wallbe is an api.ChargeController implementation for Wallbe wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
//
// This implementation will use the overcurrent protection to disable charging
// when charger is disabled or desired maxcurrent is 0A. This will lead to
// an overcurrent error state and charger LED will turn red.
//
// Upon setting a differnt, non-zero current, over overcurrent protection is
// disabled if current was equal 0A at this time.
type Wallbe struct {
	log               *api.Logger
	client            modbus.Client
	handler           *modbus.TCPClientHandler
	overchargeProtect bool
}

// NewWallbe creates a Wallbe charger
func NewWallbe(conn string) api.Charger {
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

	// init overcharge value
	if op, err := wb.overChargeEnabled(); err != nil {
		wb.log.ERROR.Printf("init overcharge protect: %v", err)
	} else {
		wb.overchargeProtect = op
	}

	return wb
}

func (wb *Wallbe) showIOs() {
	// inputs
	wb.showIO("LD", 520, 200)
	wb.showIO("EN", 521, 201)
	wb.showIO("ML", 522, 202)
	wb.showIO("XR", 523, 203)
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

func (wb *Wallbe) showIO(input string, definition uint16, status uint16) {
	var def uint16
	var val byte

	if b, err := wb.client.ReadHoldingRegisters(definition, 1); err != nil {
		wb.log.FATAL.Printf("%s definition %v", input, err)
	} else {
		def = binary.BigEndian.Uint16(b)
	}

	if b, err := wb.client.ReadDiscreteInputs(status, 1); err != nil {
		wb.log.FATAL.Printf("%s status %v", input, err)
	} else {
		val = b[0]
	}

	wb.log.DEBUG.Printf("%s = %d (%d)", input, val, def)
}

// Status implements the Charger.Status interface
func (wb *Wallbe) Status() (api.ChargeStatus, error) {
	b, err := wb.client.ReadInputRegisters(regStatus, 1)
	wb.log.TRACE.Printf("read status (%d): %0 X", regStatus, b)
	if err != nil {
		wb.handler.Close()
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *Wallbe) Enabled() (bool, error) {
	return true, nil
}

// Enable implements the Charger.Enable interface
func (wb *Wallbe) Enable(enable bool) error {
	if !enable {
		return wb.MaxCurrent(0) // set max current to zero
	}

	return nil
}

// ActualCurrent implements the Charger.ActualCurrent interface
func (wb *Wallbe) ActualCurrent() (int64, error) {
	b, err := wb.client.ReadHoldingRegisters(regActualCurrent, 1)
	wb.log.TRACE.Printf("read actual current (%d): %0 X", regActualCurrent, b)
	if err != nil {
		wb.handler.Close()
		return 0, err
	}

	u := binary.BigEndian.Uint16(b)
	return int64(u / 10), nil
}

// overChargeEnabled reads respective coil. Overcharge is only enabled
// when setting max current to 0 to disable the wallbox.
func (wb *Wallbe) overChargeEnabled() (bool, error) {
	b, err := wb.client.ReadCoils(regOverchargeProtect, 1)
	wb.log.TRACE.Printf("read overcharge protect (%d): %0 X", regOverchargeProtect, b)
	if err != nil {
		wb.handler.Close()
		return false, err
	}

	return b[0] == 1, nil
}

// overChargeEnable sets overcharge coil
func (wb *Wallbe) overChargeEnable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	b, err := wb.client.WriteSingleCoil(regOverchargeProtect, u)
	wb.log.TRACE.Printf("write overcharge protect %d %0X: %0 X", regOverchargeProtect, u, b)
	if err != nil {
		wb.handler.Close()
	} else {
		wb.overchargeProtect = enable
	}

	return err
}

// maxCurrent sets max current
func (wb *Wallbe) maxCurrent(current int64) error {
	u := uint16(current * 10)

	b, err := wb.client.WriteSingleRegister(regMaxCurrent, u)
	wb.log.TRACE.Printf("write max current %d %0X: %0 X", regMaxCurrent, u, b)
	if err != nil {
		wb.handler.Close()
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
// Setting current to 0 will always enable overcharge protection.
// If EV draw current in this state, the charger will go into error state E.
// In this state, charger will no longer notice car disconnecting (state A).
func (wb *Wallbe) MaxCurrent(current int64) error {
	var err error
	if current == 0 && !wb.overchargeProtect {
		err = wb.overChargeEnable(true)
	} else if current > 0 && wb.overchargeProtect {
		err = wb.overChargeEnable(false)
	}

	if err == nil {
		err = wb.maxCurrent(current)
	}

	return err
}

// ChargingTime yields current charge run duration
func (wb *Wallbe) ChargingTime() (time.Duration, error) {
	b, err := wb.client.ReadInputRegisters(regChargeTime, 2)
	wb.log.TRACE.Printf("read charge time (%d): %0 X", regChargeTime, b)
	if err != nil {
		wb.handler.Close()
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}
