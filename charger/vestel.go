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
	vestelSlaveID = 255
// todo Alive Register 6000 Failsafe Timeout 2002 Failsafe Current 2000
	vestelRegStatus        = 100 // Input
	vestelRegChargeTime    = 1508 // Input
	vestelRegActualCurrent = 5004 // Holding
	//vestelRegEnable        = 400 // Coil todo does not exist, but set actualCurrent to 0?
	vestelRegMaxCurrent    = 1104 // todo vestel this is not writeable! only read! Holding 0.1A
	vestelRegFirmware      = 230 // Firmware 230-279  50 register!

	vestelRegPower  = 1020 // power reading todo vestel are two registers 1020,1021!
	vestelRegEnergy = 1502 // todo energy reading vestel is Wh! Wallbe is kWh!

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

	vl, err := NewVestel(cc.URI)
	if err != nil {
		return nil, err
	}

	if cc.Legacy {
		vl.factor = 1
	}

	var currentPower func() (float64, error)
	if cc.Meter.Power {
		currentPower = vl.currentPower
	}

	var totalEnergy func() (float64, error)
	if cc.Meter.Energy {
		totalEnergy = vl.totalEnergy
	}

	var currents func() (float64, float64, float64, error)
	if cc.Meter.Currents {
		currents = vl.currents
	}

	var maxCurrentMillis func(float64) error
	if !cc.Legacy {
		maxCurrentMillis = vl.maxCurrentMillis
	}

	// special case for SDM meters
	if encoding := strings.ToLower(cc.Meter.Encoding); strings.HasPrefix(encoding, encodingSDM) {
		vl.encoding = encodingSDM
	}

	return decorateVestel(vl, currentPower, totalEnergy, currents, maxCurrentMillis), nil
}

// NewVestel creates a Vestel charger
func NewVestel(uri string) (*Vestel, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, false, vestelSlaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("vestel")
	conn.Logger(log.TRACE)

	vl := &Vestel{
		conn:   conn,
		factor: 10,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (vl *Vestel) Status() (api.ChargeStatus, error) {
	b, err := vl.conn.ReadInputRegisters(vestelRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the api.Charger interface
func (vl *Vestel) Enabled() (bool, error) {
	b, err := vl.conn.ReadHoldingRegisters(vestelRegActualCurrent, 1)
	if err != nil {
		return false, err
	}
// todo not bit but a value it is!
	return b[0] > 0, nil
}

// Enable implements the api.Charger interface
func (vl *Vestel) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := vl.conn.WriteSingleCoil(vestelRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (vl *Vestel) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current * vl.factor)
	_, err := vl.conn.WriteSingleRegister(vestelRegMaxCurrent, u)

	return err
}

// maxCurrentMillis implements the api.ChargerEx interface
func (vl *Vestel) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	u := uint16(current * float64(vl.factor))
	_, err := vl.conn.WriteSingleRegister(vestelRegMaxCurrent, u) // todo vestel: register vestelRegActualCurrent instead! contains current current and allows to set it too

	return err
}

var _ api.ChargeTimer = (*Vestel)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (vl *Vestel) ChargingTime() (time.Duration, error) {
	b, err := vl.conn.ReadInputRegisters(vestelRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}

func (vl *Vestel) decodeReading(b []byte) float64 {
	v := binary.BigEndian.Uint32(b)

	// assuming high register first
	if vl.encoding == encodingSDM {
		bits := uint32(b[3])<<0 | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
		f := math.Float32frombits(bits)
		return float64(f)
	}

	return float64(v)
}

// currentPower implements the api.Meter interface
func (vl *Vestel) currentPower() (float64, error) {
	b, err := vl.conn.ReadInputRegisters(vestelRegPower, 2)
	if err != nil {
		return 0, err
	}

	return vl.decodeReading(b), err
}

// totalEnergy implements the api.MeterEnergy interface
func (vl *Vestel) totalEnergy() (float64, error) {
	b, err := vl.conn.ReadInputRegisters(vestelRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return vl.decodeReading(b), err
}

// currents implements the api.MeterCurrent interface
func (vl *Vestel) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range vestelRegCurrents {
		b, err := vl.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, vl.decodeReading(b))
	}

	return currents[0], currents[1], currents[2], nil
}

// Diagnose implements the Diagnosis interface
func (vl *Vestel) Diagnose() {
	if b, err := vl.conn.ReadInputRegisters(vestelRegFirmware, 50); err == nil {
		fmt.Printf("Firmware:\t%s\n", encoding.StringSwapped(b))
	}
}
