package charger

import (
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Obo charger implementation
type Obo struct {
	conn *modbus.Connection
}

const (
	oboRegEnable     = 40006
	oboRegAmpsConfig = 40007
	oboRegStatus     = 40012
)

func init() {
	registry.Add("obo", NewOboFromConfig)
}

// NewOboFromConfig creates a OBO Bettermann charger from generic config
func NewOboFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewObo(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)

}

// NewObo creates OBO Bettermann charger
func NewObo(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("obo")
	conn.Logger(log.TRACE)

	wb := &Obo{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Obo) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(oboRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch r := 'A' + rune(b[1]); r {
	case 'A', 'B', 'C', 'D':
		return api.ChargeStatus(r), nil
	case 'E':
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", binary.BigEndian.Uint16(b))
	}
}

// Enabled implements the api.Charger interface
func (wb *Obo) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(oboRegEnable, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Obo) Enable(enable bool) error {
	// b := make([]byte, 2)
	var u uint16
	if enable {
		// binary.BigEndian.PutUint16(b, 1)
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(oboRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Obo) MaxCurrent(current int64) error {
	_, err := wb.conn.WriteSingleRegister(oboRegAmpsConfig, uint16(current))
	return err
}
