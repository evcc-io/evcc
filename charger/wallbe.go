package charger

import (
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

const (
	wbSlaveID = 255

	wbRegStatus     = 100 // Input
	wbRegChargeTime = 102 // Input
	wbRegEnable     = 400 // Coil
	wbRegMaxCurrent = 528 // Holding
	wbRegFirmware   = 149 // Firmware

	wbRegPower          = 120 // power reading
	wbRegEnergy         = 128 // energy reading
	wbRegEnergyDecimals = 904 // energy reading decimals
)

var wbRegCurrents = []uint16{114, 116, 118} // current readings

// Wallbe is an api.Charger implementation for Wallbe wallboxes.
// It supports both wallbe controllers (post 2019 models) and older ones using the
// Phoenix EV-CC-AC1-M3-CBC-RCM-ETH controller.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Wallbe struct {
	conn   *modbus.Connection
	factor int64
}

func init() {
	registry.Add("wallbe", NewWallbeFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateWallbe -b *Wallbe -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.ChargerEx,MaxCurrentMillis,func(current float64) error"

// NewWallbeFromConfig creates a Wallbe charger from generic config
func NewWallbeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI    string
		Legacy bool
		Meter  struct {
			Power, Energy, Currents bool
		}
	}{
		URI: "192.168.0.8:502",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWallbe(cc.URI)
	if err != nil {
		return nil, err
	}

	if cc.Legacy {
		wb.factor = 1
	}

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

	var maxCurrentMillis func(float64) error
	if !cc.Legacy {
		maxCurrentMillis = wb.maxCurrentMillis
	}

	return decorateWallbe(wb, currentPower, totalEnergy, currents, maxCurrentMillis), nil
}

// NewWallbe creates a Wallbe charger
func NewWallbe(uri string) (*Wallbe, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, wbSlaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("wallbe")
	conn.Logger(log.TRACE)

	wb := &Wallbe{
		conn:   conn,
		factor: 10,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Wallbe) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(string(b[1]))
}

// Enabled implements the api.Charger interface
func (wb *Wallbe) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(wbRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Wallbe) Enable(enable bool) error {
	var u uint16
	if enable {
		u = modbus.CoilOn
	}

	_, err := wb.conn.WriteSingleCoil(wbRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Wallbe) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current * wb.factor)
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u)

	return err
}

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *Wallbe) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	u := uint16(current * float64(wb.factor))
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u)

	return err
}

// currentPower implements the api.Meter interface
func (wb *Wallbe) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUInt32ToFloat64Swapped(b), nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Wallbe) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	res := rs485.RTUUint32ToFloat64Swapped(b)

	d, err := wb.conn.ReadHoldingRegisters(wbRegEnergyDecimals, 1)
	if err != nil {
		return 0, err
	}

	res += float64(binary.BigEndian.Uint16(d)) / 1e3

	return res, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *Wallbe) currents() (float64, float64, float64, error) {
	var res [3]float64
	for i, reg := range wbRegCurrents {
		b, err := wb.conn.ReadInputRegisters(reg, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = rs485.RTUInt32ToFloat64Swapped(b)
	}

	return res[0], res[1], res[2], nil
}

var _ api.Diagnosis = (*Wallbe)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Wallbe) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(wbRegFirmware, 6); err == nil {
		fmt.Printf("Firmware:\t%s\n", encoding.StringLsbFirst(b))
	}
}
