package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

const (
	phEVCCRegEnable     = 20000 // Coil
	phEVCCRegMaxCurrent = 22000 // Holding
	phEVCCRegStatus     = 24000 // Input
)

// PhoenixEVCC is an api.ChargeController implementation for Phoenix EV-CC-AC1-M wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type PhoenixEVCC struct {
	conn *modbus.Connection
}

func evccDefaults() modbus.Settings {
	return modbus.Settings{
		URI: "192.168.0.8:502", // default
		ID:  255,               // default
	}
}

func init() {
	registry.Add("phoenix-evcc", "Phoenix EV-CC", NewPhoenixEVCCFromConfig, nil)

	// TCP
	registry.Add("phoenix-evcc-tcp", "Phoenix EV-CC (TCP)", NewPhoenixEVCCFromConfig, struct {
		URI string `validate:"required"`
		ID  uint8  `ui:"de=ModBus Slave ID"`
	}{
		URI: "192.168.0.8:502", // default
		ID:  255,
	})

	// Serial
	registry.Add("phoenix-evcc-serial", "Phoenix EV-CC (Seriell)", NewPhoenixEVCCFromConfig, struct {
		Device   string `validate:"required" ui:"de=Serielle Schnittstelle"`
		Comset   string `validate:"required,oneof=8E1 8N1" ui:"de=Kommunikationseinstellungen"`
		Baudrate int    `validate:"required,oneof=2400 9600 19200" ui:"de=Baudrate"`
		ID       uint8  `ui:"de=ModBus Slave ID"`
	}{
		Device:   "/dev/usb0",
		Comset:   "8N1",
		Baudrate: 9600,
		ID:       255,
	})
}

// NewPhoenixEVCCFromConfig creates a Phoenix charger from generic config
func NewPhoenixEVCCFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := evccDefaults()

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPhoenixEVCC(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

// NewPhoenixEVCC creates a Phoenix charger
func NewPhoenixEVCC(uri, device, comset string, baudrate int, id uint8) (*PhoenixEVCC, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, true, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("evcc")
	conn.Logger(log.TRACE)

	wb := &PhoenixEVCC{
		conn: conn,
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *PhoenixEVCC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phEVCCRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[0])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixEVCC) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phEVCCRegEnable, 1)
	if err != nil {
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

	_, err := wb.conn.WriteSingleCoil(phEVCCRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixEVCC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phEVCCRegMaxCurrent, uint16(current))

	return err
}
