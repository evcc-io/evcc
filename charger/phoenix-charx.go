package charger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
)

const (
	// holding and input return same values
	charxRegName           = 100
	charxRegSwVersion      = 110
	charxRegNumControllers = 114

	// per-unit registers
	charxRegMeter = 112

	charxRegVoltages     = 232 // mV
	charxRegCurrents     = 238 // mA
	charxRegPower        = 244 // mW
	charxRegEnergy       = 250 // Wh
	charxRegSoc          = 264 // %
	charxRegEvid         = 265 // 10
	charxRegRfid         = 275 // 10
	charxRegChargeTime   = 287 // s
	charxRegChargeEnergy = 289 // Wh

	charxRegStatus     = 299 // IEC 61851-1
	charxRegEnable     = 300
	charxRegMaxCurrent = 301 // A
)

// PhoenixCharx is an api.Charger implementation for Phoenix CHARX controller
type PhoenixCharx struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-charx", NewPhoenixCharxFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePhoenixCharx -b *PhoenixCharx -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewPhoenixCharxFromConfig creates a Phoenix charger from generic config
func NewPhoenixCharxFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1, // default
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewPhoenixCharx(cc.URI, cc.ID)

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

	return decoratePhoenixCharx(wb, currentPower, totalEnergy, currents), err
}

// NewPhoenixCharx creates a Phoenix charger
func NewPhoenixCharx(uri string, id uint8) (*PhoenixCharx, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("charx")
	conn.Logger(log.TRACE)

	wb := &PhoenixCharx{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *PhoenixCharx) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(charxRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the api.Charger interface
func (wb *PhoenixCharx) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(charxRegEnable, 1)
	if err != nil {
		return false, err
	}

	return encoding.Uint16(b) == 1, nil
}

// Enable implements the api.Charger interface
func (wb *PhoenixCharx) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(charxRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PhoenixCharx) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(charxRegMaxCurrent, uint16(current))

	return err
}

var _ api.ChargeTimer = (*PhoenixCharx)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *PhoenixCharx) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(charxRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint16(b)) * time.Second, nil
}

// CurrentPower implements the api.Meter interface
func (wb *PhoenixCharx) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(charxRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32(b)) / 1e3, nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *PhoenixCharx) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(charxRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	return encoding.Float64(b) / 1e3, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *PhoenixCharx) currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(charxRegCurrents, 3*2)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(encoding.Float32(b[4*i:])) / 1e3
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseVoltages = (*PhoenixCharx)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *PhoenixCharx) Voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(charxRegVoltages, 3*2)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(encoding.Float32(b[4*i:])) / 1e3
	}

	return res[0], res[1], res[2], nil
}

var _ api.Diagnosis = (*PhoenixCharx)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *PhoenixCharx) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(charxRegName, 10); err == nil {
		fmt.Printf("Name: %s\n", encoding.StringLsbFirst(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(charxRegSwVersion, 4); err == nil {
		fmt.Printf("Software version: %s\n", encoding.StringLsbFirst(b))
	}

	var controllers uint16
	if b, err := wb.conn.ReadHoldingRegisters(charxRegNumControllers, 1); err == nil {
		controllers = binary.BigEndian.Uint16(b)
		fmt.Printf("Controllers: %d\n", controllers)
	}
}
