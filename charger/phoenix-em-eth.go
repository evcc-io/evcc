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
	phxEMEthRegStatus     = 100 // Input
	phxEMEthRegChargeTime = 102 // Input
	phxEMEthRegMaxCurrent = 300 // Holding
	phxEMEthRegEnable     = 400 // Coil

	phxEMEthRegPower  = 120 // power reading
	phxEMEthRegEnergy = 128 // energy reading
)

var phxEMEthRegCurrents = []uint16{114, 116, 118} // current readings

// PhoenixEMEth is an api.Charger implementation for Phoenix EM-CP-PP-ETH wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 180.
type PhoenixEMEth struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-em-eth", NewPhoenixEMEthFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePhoenixEMEth -b *PhoenixEMEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewPhoenixEMEthFromConfig creates a Phoenix charger from generic config
func NewPhoenixEMEthFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		ID    uint8
		Meter struct {
			Power, Energy, Currents bool
		}
	}{
		URI: "192.168.0.8:502", // default
		ID:  180,               // default
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPhoenixEMEth(cc.URI, cc.ID)

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

	return decoratePhoenixEMEth(wb, currentPower, totalEnergy, currents), err
}

// NewPhoenixEMEth creates a Phoenix charger
func NewPhoenixEMEth(uri string, id uint8) (*PhoenixEMEth, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("em-eth")
	conn.Logger(log.TRACE)

	wb := &PhoenixEMEth{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *PhoenixEMEth) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxEMEthRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the api.Charger interface
func (wb *PhoenixEMEth) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxEMEthRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the api.Charger interface
func (wb *PhoenixEMEth) Enable(enable bool) error {
	var u uint16
	if enable {
		u = modbus.CoilOn
	}

	_, err := wb.conn.WriteSingleCoil(phxEMEthRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PhoenixEMEth) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phxEMEthRegMaxCurrent, uint16(current))

	return err
}

var _ api.ChargeTimer = (*PhoenixEMEth)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *PhoenixEMEth) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(phxEMEthRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(secs) * time.Second, nil
}

// CurrentPower implements the api.Meter interface
func (wb *PhoenixEMEth) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxEMEthRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) * 10, err
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *PhoenixEMEth) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxEMEthRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 100, err
}

// currents implements the api.PhaseCurrents interface
func (wb *PhoenixEMEth) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range phxEMEthRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, rs485.RTUUint32ToFloat64Swapped(b)/1000)
	}

	return currents[0], currents[1], currents[2], nil
}
