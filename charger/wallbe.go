package charger

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

const (
	wbSlaveID = 255

	wbRegStatus        = 100 // Input
	wbRegChargeTime    = 102 // Input
	wbRegActualCurrent = 300 // Holding
	wbRegEnable        = 400 // Coil
	wbRegMaxCurrent    = 528 // Holding

	wbRegPower  = 120 // power reading
	wbRegEnergy = 128 // energy reading

	encodingSDM = "sdm"
)

var wbRegCurrents = []uint16{114, 116, 118} // current readings

// Wallbe is an api.ChargeController implementation for Wallbe wallboxes.
// It supports both wallbe controllers (post 2019 models) and older ones using the
// Phoenix EV-CC-AC1-M3-CBC-RCM-ETH controller.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Wallbe struct {
	conn     *modbus.Connection
	factor   int64
	encoding string
}

func init() {
	registry.Add("wallbe", NewWallbeFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -p charger -f decorateWallbe -b api.Charger -o wallbe_decorators -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.ChargerEx,MaxCurrentMillis,func(current float64) error"

// NewWallbeFromConfig creates a Wallbe charger from generic config
func NewWallbeFromConfig(other map[string]interface{}) (api.Charger, error) {
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

	// special case for SDM meters
	if encoding := strings.ToLower(cc.Meter.Encoding); strings.HasPrefix(encoding, encodingSDM) {
		wb.encoding = encodingSDM
	}

	return decorateWallbe(wb, currentPower, totalEnergy, currents, maxCurrentMillis), nil
}

// NewWallbe creates a Wallbe charger
func NewWallbe(uri string) (*Wallbe, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, false, wbSlaveID)
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

// Status implements the Charger.Status interface
func (wb *Wallbe) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *Wallbe) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(wbRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *Wallbe) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := wb.conn.WriteSingleCoil(wbRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *Wallbe) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current * wb.factor)
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u)

	return err
}

// maxCurrentMillis implements the ChargerEx interface
func (wb *Wallbe) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	u := uint16(current * float64(wb.factor))
	_, err := wb.conn.WriteSingleRegister(wbRegMaxCurrent, u)

	return err
}

// ChargingTime yields current charge run duration
func (wb *Wallbe) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}

func (wb *Wallbe) decodeReading(b []byte) float64 {
	v := binary.BigEndian.Uint32(b)

	// assuming high register first
	if wb.encoding == encodingSDM {
		bits := uint32(b[3])<<0 | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
		f := math.Float32frombits(bits)
		return float64(f)
	}

	return float64(v)
}

// currentPower implements the Meter.CurrentPower interface
func (wb *Wallbe) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegPower, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// totalEnergy implements the Meter.TotalEnergy interface
func (wb *Wallbe) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wbRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return wb.decodeReading(b), err
}

// currents implements the Meter.Currents interface
func (wb *Wallbe) currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range wbRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, wb.decodeReading(b))
	}

	return currents[0], currents[1], currents[2], nil
}
