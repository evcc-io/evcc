package charger

import (
	"encoding/binary"
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

// ABLeMH charger implementation
type ABLeMH struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
}

const (
	ablRegFirmware      = 0x01
	ablRegVehicleStatus = 0x04
	ablRegAmpsConfig    = 0x14
)

func init() {
	registry.Add("abl", NewABLeMHFromConfig)
}

// https://www.goingelectric.de/forum/viewtopic.php?p=1550459#p1550459

// NewABLeMHFromConfig creates a ABLeMH charger from generic config
func NewABLeMHFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
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
		log:     log,
		conn:    conn,
		current: 60, // assume min current
	}

	return wb, nil
}

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

// Enabled implements the api.Charger interface
func (wb *ABLeMH) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(ablRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	cur := binary.BigEndian.Uint16(b)

	enabled := cur != 0
	if enabled {
		wb.current = cur
	}

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *ABLeMH) Enable(enable bool) error {
	var cur uint16
	if enable {
		cur = wb.current
	}

	_, err := wb.conn.WriteSingleRegister(ablRegAmpsConfig, cur)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *ABLeMH) MaxCurrent(current int64) error {
	// 01 10 00 1400 0102 0064
	b := []byte{0x01, 0x02}
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

	_, err := wb.conn.WriteMultipleRegisters(ablRegAmpsConfig, 2, b)

	return err
}

var _ api.Diagnosis = (*ABLeMH)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ABLeMH) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(ablRegFirmware, 2); err == nil {
		fmt.Printf("Firmware: %0 x\n", b)
	}
}
