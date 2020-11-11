package charger

import (
	"encoding/binary"
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

const (
	phEVCCRegStatus     = 24000 // Input
	phEVCCRegMaxCurrent = 22000 // Holding
	phEVCCRegEnable     = 20000 // Coil

	phRegPower  = 337 // power reading
	phRegEnergy = 341 // energy reading
)

var phRegCurrents = []uint16{334, 335, 336} // current readings

// PhoenixEVCC is an api.ChargeController implementation for Phoenix EV-CC-AC1-M wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type PhoenixEVCC struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-evcc", NewPhoenixEVCCFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -p charger -f decoratePhoenixEVCC -b api.Charger -o phoenix-evcc_decorators -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewPhoenixEVCCFromConfig creates a Phoenix charger from generic config
func NewPhoenixEVCCFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Meter           struct {
			Power, Energy, Currents bool
		}
	}{
		Settings: modbus.Settings{
			ID: 255,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPhoenixEVCC(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)

	var currentPower func() (float64, error)
	if cc.Meter.Power {
		currentPower = wb.currentPower
	}

	var totalEnergy func() (float64, error)
	if cc.Meter.Energy {
		totalEnergy = wb.totalEnergy
	}

	var currents func() (float64, float64, float64, error)
	if cc.Meter.Currents {
		currents = wb.currents
	}

	return decoratePhoenixEVCC(wb, currentPower, totalEnergy, currents), err
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

func (wb *PhoenixEVCC) decodeReading(b []byte) float64 {
	v := binary.BigEndian.Uint16(b)
	return float64(v)
}

// CurrentPower implements the Meter.CurrentPower interface
func (wb *PhoenixEVCC) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(phRegPower, 1)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// totalEnergy implements the Meter.TotalEnergy interface
func (wb *PhoenixEVCC) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(phRegEnergy, 1)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// currents implements the Meter.Currents interface
func (wb *PhoenixEVCC) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range phRegCurrents {
		b, err := wb.conn.ReadHoldingRegisters(regCurrent, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, wb.decodeReading(b))
	}

	return currents[0], currents[1], currents[2], nil
}
