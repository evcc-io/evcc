package charger

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
)

const (
	phxEMEthRegStatus     = 100 // Input
	phxEMEthRegChargeTime = 102 // Input
	phxEMEthRegVoltages   = 108 // Input
	phxEMEthRegCurrents   = 114 // Input
	phxEMEthRegPower      = 120 // Input
	phxEMEthRegEnergy     = 128 // Input
	phxEMEthRegMaxCurrent = 300 // Holding
	phxEMEthRegEnable     = 400 // Coil
)

// PhoenixEMEth is an api.Charger implementation for Phoenix EM-CP-PP-ETH wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 180.
type PhoenixEMEth struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-em-eth", NewPhoenixEMEthFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePhoenixEMEth -b *PhoenixEMEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)"

// NewPhoenixEMEthFromConfig creates a Phoenix charger from generic config
func NewPhoenixEMEthFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 180,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPhoenixEMEth(cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	var (
		currentPower func() (float64, error)
		totalEnergy  func() (float64, error)
		currents     func() (float64, float64, float64, error)
		voltages     func() (float64, float64, float64, error)
	)

	// check presence of meter by voltage on l1
	if b, err := wb.conn.ReadInputRegisters(phxEMEthRegVoltages, 2); err == nil && encoding.Int32LswFirst(b) > 0 {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
		currents = wb.currents
		voltages = wb.voltages
	}

	return decoratePhoenixEMEth(wb, currentPower, totalEnergy, currents, voltages), err
}

// NewPhoenixEMEth creates a Phoenix charger
func NewPhoenixEMEth(uri string, slaveID uint8) (*PhoenixEMEth, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
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

	return api.ChargeStatusString(string(b[1]))
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

	return float64(encoding.Int32LswFirst(b)) * 10, err
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *PhoenixEMEth) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxEMEthRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int32LswFirst(b)) / 100, err
}

// currents implements the api.PhaseCurrents interface
func (wb *PhoenixEMEth) currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(phxEMEthRegCurrents)
}

// voltages implements the api.PhaseVoltages interface
func (wb *PhoenixEMEth) voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(phxEMEthRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *PhoenixEMEth) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(encoding.Int32LswFirst(b[4*i:])) / 1e3
	}

	return res[0], res[1], res[2], nil
}

var _ api.CurrentGetter = (*PhoenixEMEth)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb PhoenixEMEth) GetMaxCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(phxEMEthRegMaxCurrent, 1)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint16(b)), err
}
