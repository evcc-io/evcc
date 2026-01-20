package charger

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Voltie charger implementation
// https://voltie.eu
// Modbus API documentation v1.02

const (
	voltieRegChargerID       = 0x0000 // R, INT16, Voltie Charger ID
	voltieRegFirmware        = 0x0001 // R, INT16, FW version
	voltieRegStatus          = 0x000A // R, INT16, EVSE_STATE
	voltieRegAutoStart       = 0x000B // R/W, INT16, Auto Start enabled
	voltieRegChargingEnabled = 0x000C // R/W, INT16, Charging enabled
	voltieRegCharging        = 0x000D // R, INT16, Charging (0=no charging, 1=charging)
	voltieRegPhases          = 0x000E // R, INT16, Number of phases in use
	voltieRegStopReason      = 0x0012 // R, INT16, Charge stop reason
	voltieRegCurrentLimit    = 0x0014 // R/W, INT16, Software current limit [mA]

	voltieRegVoltageL1      = 0x2000 // R, INT32, Phase L1 voltage [mV]
	voltieRegVoltageL2      = 0x2002 // R, INT32, Phase L2 voltage [mV]
	voltieRegVoltageL3      = 0x2004 // R, INT32, Phase L3 voltage [mV]
	voltieRegCurrentL1      = 0x2006 // R, INT32, Phase L1 charging current [mA]
	voltieRegCurrentL2      = 0x2008 // R, INT32, Phase L2 charging current [mA]
	voltieRegCurrentL3      = 0x200A // R, INT32, Phase L3 charging current [mA]
	voltieRegChargeDuration = 0x200C // R, INT32, Charge duration [s]
	voltieRegChargedEnergy  = 0x200E // R, INT32, Charged energy in current session [Ws]
	voltieRegChargingPower  = 0x2010 // R, INT32, Charging power [W]
)

// Voltie is an api.Charger implementation for Voltie wallboxes
type Voltie struct {
	conn *modbus.Connection
}

func init() {
	registry.AddCtx("voltie", NewVoltieFromConfig)
}

//go:generate go tool decorate -f decorateVoltie -b *Voltie -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewVoltieFromConfig creates a Voltie charger from generic config
func NewVoltieFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Meter              struct {
			Power, Energy, Currents bool
		}
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewVoltie(ctx, cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	// decorate meter
	var (
		power, energy func() (float64, error)
		currents      func() (float64, float64, float64, error)
	)

	if cc.Meter.Power {
		power = wb.currentPower
	}

	if cc.Meter.Energy {
		energy = wb.totalEnergy
	}

	if cc.Meter.Currents {
		currents = wb.currents
	}

	return decorateVoltie(wb, power, energy, currents), nil
}

// NewVoltie creates a Voltie charger
func NewVoltie(ctx context.Context, uri string, slaveID uint8) (*Voltie, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("voltie")
	conn.Logger(log.TRACE)

	wb := &Voltie{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Voltie) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	status := binary.BigEndian.Uint16(b)

	// EVSE states:
	// 0x01: vehicle in state A – not connected
	// 0x02: vehicle in state B – connected, ready
	// 0x03: vehicle in state C – charging
	// 0x04: vehicle in state D – charging, ventilation required
	// 0x0D: vehicle in state E – vehicle error
	// 0x05-0x0C, 0x0E-0x11: internal error states
	// 0xFF: charger disabled, not functioning

	switch status {
	case 0x01, 0xFF:
		return api.StatusA, nil
	case 0x02:
		return api.StatusB, nil
	case 0x03, 0x04:
		return api.StatusC, nil
	default:
		// 0x0D: vehicle in state E – vehicle error
		// 0x05-0x0C, 0x0E-0x11: internal error states
		return api.StatusE, nil
	}
}

// Enabled implements the api.Charger interface
func (wb *Voltie) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegChargingEnabled, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Voltie) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(voltieRegChargingEnabled, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Voltie) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	// Convert A to mA
	u := uint16(current * 1000)
	_, err := wb.conn.WriteSingleRegister(voltieRegCurrentLimit, u)

	return err
}

// currentPower implements the api.Meter interface
func (wb *Voltie) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegChargingPower, 2)
	if err != nil {
		return 0, err
	}

	// Power in W
	power := int32(binary.BigEndian.Uint32(b))
	return float64(power), nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Voltie) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	// Energy in Ws, convert to kWh
	energy := int32(binary.BigEndian.Uint32(b))
	return float64(energy) / 3600000, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *Voltie) currents() (float64, float64, float64, error) {
	var res [3]float64
	regs := []uint16{voltieRegCurrentL1, voltieRegCurrentL2, voltieRegCurrentL3}

	for i, reg := range regs {
		b, err := wb.conn.ReadHoldingRegisters(reg, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		// Current in mA, convert to A
		current := int32(binary.BigEndian.Uint32(b))
		res[i] = float64(current) / 1000
	}

	return res[0], res[1], res[2], nil
}

var _ api.Diagnosis = (*Voltie)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Voltie) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegChargerID, 1); err == nil {
		fmt.Printf("\tCharger ID:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegFirmware, 1); err == nil {
		fmt.Printf("\tFirmware:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegStatus, 1); err == nil {
		fmt.Printf("\tStatus:\t\t0x%04X\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegPhases, 1); err == nil {
		fmt.Printf("\tPhases:\t\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegStopReason, 1); err == nil {
		fmt.Printf("\tStop reason:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
