package charger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// PhoenixEVEth is an api.Charger implementation for Phoenix EV-***-ETH controller models
// EV-CC-AC1-M3-CBC-RCM-ETH, EV-CC-AC1-M3-CBC-RCM-ETH-3G, EV-CC-AC1-M3-RCM-ETH-XP, EV-CC-AC1-M3-RCM-ETH-3G-XP
// with OEM firmware from Phoenix Contact and modified firmware versions (Wallbe/Compleo).
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type PhoenixEVEth struct {
	conn     *modbus.Connection
	isWallbe bool
}

const (
	wbRegStatus          = 100  // Input
	wbRegChargeTime      = 102  // Input
	wbRegFirmware        = 105  // Input
	wbRegVoltages        = 108  // Input
	wbRegCurrents        = 114  // Input
	wbRegPower           = 120  // Input
	wbRegEnergy          = 128  // Input
	wbRegChargedEnergy   = 132  // Input
	wbRegFirmwareWallbe  = 149  // Input
	wbRegEnable          = 400  // Coil
	wbRegCardEnabled     = 419  // Coil
	wbRegMaxCurrent      = 528  // Holding
	wbRegCardUID         = 606  // Holding
	wbRegEnergyWh        = 904  // Holding, 32bit, Wh (2), Wallbe: 16bit (1)
	wbRegEnergyWallbe    = 2980 // Holding, 64bit, Wh (4)
	wbRegChargedEnergyEx = 3376 // Holding, 64bit, Wh (4)
)

func init() {
	registry.Add("phoenix-ev-eth", NewPhoenixEVEthFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePhoenixEVEth -b *PhoenixEVEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.ChargerEx,MaxCurrentMillis,func(current float64) error"

// NewPhoenixEVEthFromConfig creates a PhoenixEVEth charger from generic config
func NewPhoenixEVEthFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI string
	}{
		URI: "192.168.0.8:502",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPhoenixEVEth(cc.URI)
}

// NewPhoenixEVEth creates a PhoenixEVEth charger
func NewPhoenixEVEth(uri string) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, 255)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("ev-eth")
	conn.Logger(log.TRACE)

	wb := &PhoenixEVEth{
		conn: conn,
	}

	var (
		currentPower     func() (float64, error)
		totalEnergy      func() (float64, error)
		currents         func() (float64, float64, float64, error)
		voltages         func() (float64, float64, float64, error)
		identify         func() (string, error)
		maxCurrentMillis func(float64) error
	)

	// check presence of meter by voltage on l1
	if b, err := wb.conn.ReadInputRegisters(wbRegVoltages, 2); err == nil && binary.BigEndian.Uint32(b) >= 0 {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
		currents = wb.currents
		voltages = wb.voltages
	}

	// check card reader enabled
	if b, err := wb.conn.ReadCoils(wbRegCardEnabled, 1); err == nil && b[0] == 1 {
		identify = wb.identify
	}

	// check presence of extended Wallbe firmware
	if b, err := wb.conn.ReadInputRegisters(wbRegMaxCurrent, 1); err == nil && binary.BigEndian.Uint16(b) >= 60 {
		wb.isWallbe = true
		maxCurrentMillis = wb.maxCurrentMillis
	}

	return decoratePhoenixEVEth(wb, currentPower, totalEnergy, currents, voltages, identify, maxCurrentMillis), err
}

// Status implements the api.Charger interface
func (wb *PhoenixEVEth) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(string(b[1]))
}

// Enabled implements the api.Charger interface
func (wb *PhoenixEVEth) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(wbRegEnable, 1)
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

	_, err := wb.conn.WriteSingleCoil(wbRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PhoenixEVEth) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current)
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u)

	return err
}

// maxCurrentMillis implements the api.ChargerEx interface (Wallbe Firmware only)
func (wb *PhoenixEVEth) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	u := uint16(current * 10)
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u)

	return err
}

var _ api.ChargeTimer = (*PhoenixEVEth)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *PhoenixEVEth) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[1]) | uint64(b[0])<<8 | uint64(b[3])<<16 | uint64(b[2])<<24
	return time.Duration(secs) * time.Second, nil
}

// currentPower implements the api.Meter interface
func (wb *PhoenixEVEth) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUInt32ToFloat64Swapped(b), nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *PhoenixEVEth) totalEnergy() (float64, error) {
	if wb.isWallbe {
		b, err := wb.conn.ReadInputRegisters(wbRegEnergyWallbe, 4)
		if err != nil {
			return 0, err
		}

		res := float64(uint64(b[1]) | uint64(b[0])<<8 | uint64(b[3])<<16 | uint64(b[2])<<24) // RTUInt64ToFloat64Swapped
		return res / 1e3, nil
	}

	b, err := wb.conn.ReadInputRegisters(wbRegEnergyWh, 2)
	if err != nil {
		return 0, err
	}

	res := rs485.RTUUint32ToFloat64Swapped(b) / 1e3

	return res, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *PhoenixEVEth) currents() (float64, float64, float64, error) {
	return wb.getPhases(wbRegCurrents)
}

// voltages implements the api.PhaseVoltages interface
func (wb *PhoenixEVEth) voltages() (float64, float64, float64, error) {
	return wb.getPhases(wbRegVoltages)
}

// getPhases returns 3 sequential phase values
func (wb *PhoenixEVEth) getPhases(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = rs485.RTUInt32ToFloat64Swapped(b[2*i:])
	}

	return res[0], res[1], res[2], nil
}

// identify implements the api.Identifier interface
func (wb *PhoenixEVEth) identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(wbRegCardUID, 16)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

var _ api.Diagnosis = (*PhoenixEVEth)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *PhoenixEVEth) Diagnose() {
	if wb.isWallbe {
		if b, err := wb.conn.ReadInputRegisters(wbRegFirmwareWallbe, 6); err == nil {
			fmt.Printf("\tFirmware (Wallbe):\t%s\n", encoding.StringLsbFirst(b))
		}
	} else {
		if b, err := wb.conn.ReadInputRegisters(wbRegFirmware, 2); err == nil {
			fmt.Printf("\tFirmware (Phoenix):\t%s\n", encoding.StringLsbFirst(b))
		}
	}
}
