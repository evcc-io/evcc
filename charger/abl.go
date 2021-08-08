package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

// ABLeMH charger implementation
type ABLeMH struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	ablRegFirmware      = 0x01
	ablRegVehicleStatus = 0x04
	ablRegEnable        = 0x05
	ablRegAmpsConfig    = 0x14
)

func init() {
	registry.Add("abl", NewABLeMHFromConfig)
}

// https://www.goingelectric.de/forum/viewtopic.php?p=1550459#p1550459

// NewABLeMHFromConfig creates a ABLeMH charger from generic config
func NewABLeMHFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewABLeMH(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

// NewABLeMH creates ABLeMH charger
func NewABLeMH(uri, device, comset string, baudrate int, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, modbus.AsciiFormat, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("abl")
	conn.Logger(log.TRACE)

	wb := &ABLeMH{
		log:  log,
		conn: conn,
	}

	return wb, nil
}

//  Start           : 1 char
//  Address         : 2 chars
//  Function        : 2 chars
//  Data            : 0 up to 2x252 chars
//  LRC             : 2 chars
//  End             : 2 chars

// readHoldingRegisterFunctionCode = 0x03
// readInputRegisterFunctionCode   = 0x04

// writeSingleRegisterFunctionCode   = 0x06
// writeMultipleRegisterFunctionCode = 0x10

// Status implements the api.Charger interface
func (wb *ABLeMH) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(ablRegVehicleStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	r := rune(b[1]>>4-0x0A) + 'A'

	switch r {
	case 'A', 'B', 'C':
		return api.ChargeStatus(r), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %v", r)
	}
}

// :0F 10 0005 0001 02 02A1 36

// Enabled implements the api.Charger interface
func (wb *ABLeMH) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(ablRegEnable, 1)
	if err != nil {
		return false, err
	}

	_ = b
	enabled := true

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *ABLeMH) Enable(enable bool) error {
	b := []byte{0x02, 0xE0}
	if enable {
		b[1] = 0xA1
	}

	_, err := wb.conn.WriteMultipleRegisters(ablRegEnable, 1, b)

	return err
}

//  Start           : 1 char
//  Address         : 2 chars
//  Function        : 2 chars
//  Data            : 0 up to 2x252 chars
//  LRC             : 2 chars
//  End             : 2 chars

// :01 10 0005 0001 02 A1A1
// :01 10 0014 0001 02 0064
// :0F 10 0014 0002 02 0064 65\r\n
//  0F 10 0014 0001 02 010B BE

// MaxCurrent implements the api.Charger interface
func (wb *ABLeMH) MaxCurrent(current int64) error {
	b := []byte{}
	c := byte(current)

	switch current {
	case 6, 7, 8:
		b = append(b, 0x00, c<<4+c-2)
	case 9, 10, 11:
		b = append(b, 0x00, c<<4+c-3)
	case 12, 13, 14:
		b = append(b, 0x00, c<<4+c-4)
	case 15:
		b = append(b, 0x00, 0xFA)
	case 16:
		b = append(b, 0x01, 0x0B)
	default:
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteMultipleRegisters(ablRegAmpsConfig, 1, b)

	return err
}

var _ api.Diagnosis = (*ABLeMH)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ABLeMH) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(ablRegFirmware, 2); err == nil {
		fmt.Printf("Firmware: %0 x\n", b)
	}
}
