package charger

// todo vestel hymes

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
)

const (
	wbSlaveID = 255
// todo Alive Register 6000 Failsafe Timeout 2002 Failsafe Current 2000
	wbRegStatus        = 100 // Input
	wbRegChargeTime    = 1508 // Input
	wbRegActualCurrent = 5004 // Holding
	//wbRegEnable        = 400 // Coil todo does not exist, but set actualCurrent to 0?
	wbRegMaxCurrent    = 1104 // todo vestel this is not writeable! only read! Holding 0.1A
	wbRegFirmware      = 230 // Firmware 230-279  50 register!

	wbRegPower  = 1020 // power reading 1020,1021
	wbRegEnergy = 1502 // todo energy reading vestel is Wh! Wallbe is kWh!

	encodingSDM = "sdm"
)

var vestelRegCurrents = []uint16{1008, 1010, 1012} // current readings // todo this are mA! wallbe is A!

// Vestel is an api.ChargeController implementation for Vestel/Hymes wallboxes with Ethernet (SW modells).
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Vestel struct {
	conn     *modbus.Connection
	factor   int64
	encoding string
}

func init() {
	registry.Add("vestel", NewVestelFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateVestel -b *Vestel -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.ChargerEx,MaxCurrentMillis,func(current float64) error"

// NewVestelFromConfig creates a Vestel charger from generic config
func NewVestelFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI    string
		Legacy bool
		Meter  struct {
			Power, Energy, Currents bool
			Encoding                string
		}
	}{
		URI: "192.168.0.8:502",
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewVestel(cc.URI)
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

	// special case for SDM meters
	if encoding := strings.ToLower(cc.Meter.Encoding); strings.HasPrefix(encoding, encodingSDM) {
		wb.encoding = encodingSDM
	}

	return decorateVestel(wb, currentPower, totalEnergy, currents, maxCurrentMillis), nil
}

// NewVestel creates a Vestel charger
func NewVestel(uri string) (*Vestel, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, false, wbSlaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("vestel")
	conn.Logger(log.TRACE)

	wb := &Vestel{
		conn:   conn,
		factor: 10,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Vestel) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the api.Charger interface
func (wb *Vestel) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wbRegActualCurrent, 1)
	if err != nil {
		return false, err
	}
// todo not bit but a value it is!
	return b[0] > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Vestel) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := wb.conn.WriteSingleCoil(wbRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Vestel) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current * wb.factor)
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u)

	return err
}

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *Vestel) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	u := uint16(current * float64(wb.factor))
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u) // todo vestel: register wbRegActualCurrent instead! contains current current and allows to set it too

	return err
}

var _ api.ChargeTimer = (*Vestel)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Vestel) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}

func (wb *Vestel) decodeReading(b []byte) float64 {
	v := binary.BigEndian.Uint32(b)

	// assuming high register first
	if wb.encoding == encodingSDM {
		bits := uint32(b[3])<<0 | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
		f := math.Float32frombits(bits)
		return float64(f)
	}

	return float64(v)
}

// currentPower implements the api.Meter interface
func (wb *Vestel) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegPower, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Vestel) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// currents implements the api.MeterCurrent interface
func (wb *Vestel) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range vestelRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, wb.decodeReading(b))
	}

	return currents[0], currents[1], currents[2], nil
}

// Diagnose implements the Diagnosis interface
func (wb *Vestel) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(wbRegFirmware, 50); err == nil {
		fmt.Printf("Firmware:\t%s\n", encoding.StringSwapped(b))
	}
}
