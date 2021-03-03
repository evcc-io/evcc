package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

const (
	phxEVEthRegStatus     = 100 // Input
	phxEVEthRegChargeTime = 102 // Input
	phxEVEthRegMaxCurrent = 528 // Holding
	phxEVEthRegEnable     = 400 // Coil

	phxEVEthRegPower  = 120 // power reading
	phxEVEthRegEnergy = 904 // energy reading, 128 for fw <= 1.11

	phxEVEthRegPowerScale   = 364 // power scaler
	phxEVEthRegEnergyScale  = 372 // energy scaler
	phxEVEthRegCurrentScale = 902 // current scaler, 358 for fw <= 1.11
)

var phxEVEthRegCurrents = []uint16{114, 116, 118} // current readings

// PhoenixEVEth is an api.ChargeController implementation for Phoenix EV-***-ETH controller models
// EV-CC-AC1-M3-CBC-RCM-ETH, EV-CC-AC1-M3-CBC-RCM-ETH-3G, EV-CC-AC1-M3-RCM-ETH-XP, EV-CC-AC1-M3-RCM-ETH-3G-XP
// It uses Modbus TCP to communicate with the controller at modbus client id 255.
type PhoenixEVEth struct {
	conn         *modbus.Connection
	powerScale   float64
	energyScale  float64
	currentScale float64
}

func init() {
	registry.Add("phoenix-ev-eth", NewPhoenixEVEthFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -p charger -f decoratePhoenixEVEth -o phoenix-ev-eth_decorators -b *PhoenixEVEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewPhoenixEVEthFromConfig creates a Phoenix charger from generic config
func NewPhoenixEVEthFromConfig(other map[string]interface{}) (api.Charger, error) {
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

	wb, err := NewPhoenixEVEth(cc.URI, cc.ID)

	var currentPower func() (float64, error)
	if cc.Meter.Power {
		currentPower = wb.currentPower
		wb.scaler(&wb.powerScale, phxEVEthRegPowerScale)
	}

	var totalEnergy func() (float64, error)
	if cc.Meter.Energy {
		totalEnergy = wb.totalEnergy
		wb.scaler(&wb.energyScale, phxEVEthRegEnergyScale)
	}

	var currents func() (float64, float64, float64, error)
	if cc.Meter.Currents {
		currents = wb.currents
		wb.scaler(&wb.currentScale, phxEVEthRegCurrentScale)
	}

	return decoratePhoenixEVEth(wb, currentPower, totalEnergy, currents), err
}

// NewPhoenixEVEth creates a Phoenix charger
func NewPhoenixEVEth(uri string, id uint8) (*PhoenixEVEth, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, false, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ev-eth")
	conn.Logger(log.TRACE)

	wb := &PhoenixEVEth{
		conn: conn,
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *PhoenixEVEth) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVEthRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixEVEth) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxEVEthRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *PhoenixEVEth) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := wb.conn.WriteSingleCoil(phxEVEthRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixEVEth) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phxEVEthRegMaxCurrent, uint16(current))

	return err
}

// ChargingTime yields current charge run duration
func (wb *PhoenixEVEth) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVEthRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}

// scaler reads the decimal scaler value
func (wb *PhoenixEVEth) scaler(val *float64, reg uint16) {
	if b, err := wb.conn.ReadHoldingRegisters(reg, 2); err == nil {
		*val = rs485.RTUUint32ToFloat64Swapped(b)
	}

	// scaler 0 means no scaling
	if *val == 0 {
		*val = 1
	}
}

func (wb *PhoenixEVEth) decodeReading(scaler float64, b []byte) float64 {
	return scaler * rs485.RTUUint32ToFloat64Swapped(b)
}

// CurrentPower implements the Meter.CurrentPower interface
func (wb *PhoenixEVEth) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVEthRegPower, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(wb.powerScale, b), err
}

// totalEnergy implements the Meter.TotalEnergy interface
func (wb *PhoenixEVEth) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(phxEVEthRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(wb.energyScale, b) / 1e3, err
}

// currents implements the Meter.Currents interface
func (wb *PhoenixEVEth) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range phxEVEthRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, wb.decodeReading(wb.currentScale, b))
	}

	return currents[0], currents[1], currents[2], nil
}
