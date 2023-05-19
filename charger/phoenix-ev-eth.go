package charger

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

const (
	phxEVEthRegStatus     = 100 // Input
	phxEVEthRegChargeTime = 102 // Input
	phxEVEthRegMaxCurrent = 528 // Holding
	phxEVEthRegEnable     = 400 // Coil

	phxEVEthRegPower  = 120 // power reading
	phxEVEthRegEnergy = 904 // energy reading, 128 for fw <= 1.11
)

var phxEVEthRegCurrents = []uint16{114, 116, 118} // current readings

// PhoenixEVEth is an api.Charger implementation for Phoenix EV-***-ETH controller models
// EV-CC-AC1-M3-CBC-RCM-ETH, EV-CC-AC1-M3-CBC-RCM-ETH-3G, EV-CC-AC1-M3-RCM-ETH-XP, EV-CC-AC1-M3-RCM-ETH-3G-XP
// It uses Modbus TCP to communicate with the controller at modbus client id 255.
type PhoenixEVEth struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-ev-eth", NewPhoenixEVEthFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePhoenixEVEth -b *PhoenixEVEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

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
	}

	var totalEnergy func() (float64, error)
	if cc.Meter.Energy {
		totalEnergy = wb.totalEnergy
	}

	var currents func() (float64, float64, float64, error)
	if cc.Meter.Currents {
		currents = wb.currents
	}

	return decoratePhoenixEVEth(wb, currentPower, totalEnergy, currents), err
}

// NewPhoenixEVEth creates a Phoenix charger
func NewPhoenixEVEth(uri string, id uint8) (*PhoenixEVEth, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
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

// Status implements the api.Charger interface
func (wb *PhoenixEVEth) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVEthRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(string(b[1]))
}

// Enabled implements the api.Charger interface
func (wb *PhoenixEVEth) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxEVEthRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the api.Charger interface
func (wb *PhoenixEVEth) Enable(enable bool) error {
	var u uint16
	if enable {
		u = modbus.CoilOn
	}

	_, err := wb.conn.WriteSingleCoil(phxEVEthRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PhoenixEVEth) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phxEVEthRegMaxCurrent, uint16(current))

	return err
}

var _ api.ChargeTimer = (*PhoenixEVEth)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *PhoenixEVEth) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVEthRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(secs) * time.Second, nil
}

// CurrentPower implements the api.Meter interface
func (wb *PhoenixEVEth) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVEthRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b), err
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *PhoenixEVEth) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(phxEVEthRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1000, err
}

// currents implements the api.PhaseCurrents interface
func (wb *PhoenixEVEth) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range phxEVEthRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, rs485.RTUUint32ToFloat64Swapped(b))
	}

	return currents[0], currents[1], currents[2], nil
}
