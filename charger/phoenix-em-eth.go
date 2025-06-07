package charger

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

const (
	phxEMEthRegStatus        = 100 // Input
	phxEMEthRegChargeTime    = 102 // Input
	phxEMEthRegVoltages      = 108 // Input
	phxEMEthRegCurrents      = 114 // Input
	phxEMEthRegPower         = 120 // Input
	phxEMEthRegEnergy        = 128 // Input
	phxEMEthRegMaxCurrent    = 300 // Holding
	phxEMEthRegVoltagesScale = 352 // Holding
	phxEMEthRegCurrentsScale = 358 // Holding
	phxEMEthRegPowerScale    = 364 // Holding
	phxEMEthRegEnergyScale   = 372 // Holding
	phxEMEthRegEnable        = 400 // Coil
)

// PhoenixEMEth is an api.Charger implementation for Phoenix EM-CP-PP-ETH wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 180.
type PhoenixEMEth struct {
	conn *modbus.Connection
}

func init() {
	registry.AddCtx("phoenix-em-eth", NewPhoenixEMEthFromConfig)
}

//go:generate go tool decorate -f decoratePhoenixEMEth -b *PhoenixEMEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)"

// NewPhoenixEMEthFromConfig creates a Phoenix charger from generic config
func NewPhoenixEMEthFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 180,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPhoenixEMEth(ctx, cc.URI, cc.ID)
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
func NewPhoenixEMEth(ctx context.Context, uri string, slaveID uint8) (*PhoenixEMEth, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
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

// ChargeDuration implements the api.ChargeTimer interface
func (wb *PhoenixEMEth) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(phxEMEthRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Int32LswFirst(b)) * time.Second, nil
}

// CurrentPower implements the api.Meter interface
func (wb *PhoenixEMEth) readScaledValue(regValue, regScale uint16) (int32, float64, error) {
	bValue, err := wb.conn.ReadInputRegisters(regValue, 2)
	if err != nil {
		return 0, 0, err
	}
	bScale, err := wb.conn.ReadHoldingRegisters(regScale, 2)
	if err != nil {
		return 0, 0, err
	}
	scale := 1000.0 * rs485.RTUIeee754ToFloat64Swapped(bScale)

	return encoding.Int32LswFirst(bValue), scale, nil
}

func (wb *PhoenixEMEth) currentPower() (float64, error) {
	value, scale, err := wb.readScaledValue(phxEMEthRegPower, phxEMEthRegPowerScale)
	if err != nil {
		return 0, err
	}

	return float64(value) / scale, nil
}

func (wb *PhoenixEMEth) totalEnergy() (float64, error) {
	value, scale, err := wb.readScaledValue(phxEMEthRegEnergy, phxEMEthRegEnergyScale)
	if err != nil {
		return 0, err
	}

	return float64(value) / scale, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *PhoenixEMEth) currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(phxEMEthRegCurrents, phxEMEthRegCurrentsScale)
}

// voltages implements the api.PhaseVoltages interface
func (wb *PhoenixEMEth) voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(phxEMEthRegVoltages, phxEMEthRegVoltagesScale)
}

func (wb *PhoenixEMEth) readScaledValues(regValue, regScale uint16) ([]float64, error) {
	const count = 3

	bValue, err := wb.conn.ReadInputRegisters(regValue, uint16(2*count))
	if err != nil {
		return nil, err
	}
	bScale, err := wb.conn.ReadHoldingRegisters(regScale, uint16(2*count))
	if err != nil {
		return nil, err
	}

	res := make([]float64, count)
	for i := 0; i < count; i++ {
		scale := 1000.0 * rs485.RTUIeee754ToFloat64Swapped(bScale[4*i:])
		res[i] = float64(encoding.Int32LswFirst(bValue[4*i:])) / scale
	}

	return res, nil
}

func (wb *PhoenixEMEth) getPhaseValues(regValue, regScale uint16) (float64, float64, float64, error) {
	res, err := wb.readScaledValues(regValue, regScale)
	if err != nil {
		return 0, 0, 0, err
	}

	return res[0], res[1], res[2], nil
}

var _ api.CurrentGetter = (*PhoenixEMEth)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *PhoenixEMEth) GetMaxCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(phxEMEthRegMaxCurrent, 1)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint16(b)), nil
}
