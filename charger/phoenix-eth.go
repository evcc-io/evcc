package charger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

const (
	phxETHRegStatus     = 100 // Input
	phxETHRegChargeTime = 102 // Input
	phxETHRegMaxCurrent = 300 // Holding
	phxETHRegEnable     = 400 // Coil

	phxETHRegPower  = 120 // power reading
	phxETHRegEnergy = 128 // energy reading
)

var phxETHRegCurrents = []uint16{114, 116, 118} // current readings

// PhoenixEth is an api.ChargeController implementation for Phoenix Contact ETH (Ethernet) controllers.
// It uses Modbus/TCP to communicate with the controller at modbus client id 180 or 255 (default).
type PhoenixEth struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-eth", NewPhoenixEthFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -p charger -f decoratePhoenixEth -o phoenix-eth_decorators -b *PhoenixEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewPhoenixEthFromConfig creates a Phoenix charger from generic config
func NewPhoenixEthFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		ID    uint8
		Meter struct {
			Power, Energy, Currents bool
		}
	}{
		URI: "192.168.0.8:502", // default
		ID:  255,               // default
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPhoenixEth(cc.URI, cc.ID)

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

	return decoratePhoenixEth(wb, currentPower, totalEnergy, currents), err
}

// NewPhoenixEth creates a Phoenix charger
func NewPhoenixEth(uri string, id uint8) (*PhoenixEth, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, false, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("phoenix-eth")
	conn.Logger(log.TRACE)

	wb := &PhoenixEth{
		conn: conn,
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *PhoenixEth) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxETHRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixEth) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxETHRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *PhoenixEth) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := wb.conn.WriteSingleCoil(phxETHRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixEth) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phxETHRegMaxCurrent, uint16(current))

	return err
}

// ChargingTime yields current charge run duration
func (wb *PhoenixEth) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(phxETHRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}

func (wb *PhoenixEth) decodeReading(b []byte) float64 {
	v := binary.BigEndian.Uint32(b)
	return float64(v)
}

// CurrentPower implements the Meter.CurrentPower interface
func (wb *PhoenixEth) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxETHRegPower, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// totalEnergy implements the Meter.TotalEnergy interface
func (wb *PhoenixEth) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxETHRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// currents implements the Meter.Currents interface
func (wb *PhoenixEth) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range phxETHRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, wb.decodeReading(b))
	}

	return currents[0], currents[1], currents[2], nil
}
