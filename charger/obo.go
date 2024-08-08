package charger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Obo charger implementation
type Obo struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	oboRegEnable     = 5
	oboRegAmpsConfig = 6
	oboRegStatus     = 11
	oboRegTimeout    = 28
)

func init() {
	registry.Add("obo", NewOboFromConfig)
}

// NewOboFromConfig creates a OBO Bettermann charger from generic config
func NewOboFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		Baudrate: 19200,
		Comset:   "8E1",
		ID:       101,
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
		log:  log,
		conn: conn,
	}

	// get failsafe timeout from charger
	b, err := conn.ReadHoldingRegisters(oboRegTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u > 0 {
		go wb.heartbeat(time.Duration(u) * time.Millisecond / 2)
	}

	// lightshow
	// go func() {
	// 	conn.WriteSingleRegister(3, 1)
	// 	for {
	// 		for i := range res {
	// 			u := rand.Int31n(256)
	// 			conn.WriteSingleRegister(uint16(i), uint16(u))
	// 		}
	// 		time.Sleep(10 * time.Millisecond)
	// 	}
	// }()
	// time.Sleep(10 * time.Second)

	return wb, nil
}

func (wb *Obo) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.ReadHoldingRegisters(dlRegSafeCurrent, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Obo) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(oboRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch u := binary.BigEndian.Uint16(b); u {
	case 0, 1, 2:
		// A..C
		return api.ChargeStatus(string('A' + rune(u))), nil
	default:
		// D, F
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *Obo) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(oboRegEnable, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Obo) Enable(enable bool) error {
	var u uint16
	if enable {
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
