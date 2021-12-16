package charger

// todo vestel hymes

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
	// todo Alive Register 6000 Failsafe Timeout 2002 Failsafe Current 2000
	vestelRegStatus        = 100  // Input
	vestelRegChargeTime    = 1508 // Input
	vestelRegActualCurrent = 5004 // Holding
	//vestelRegEnable        = 400 // Coil todo does not exist, but set actualCurrent to 0?
	vestelRegMaxCurrent = 1104 // todo vestel this is not writeable! only read! Holding 0.1A
	vestelRegFirmware   = 230  // Firmware 230-279  50 register!

	vestelRegPower  = 1020 // power reading todo vestel are two registers 1020,1021!
	vestelRegEnergy = 1502 // todo energy reading vestel is Wh! Wallbe is kWh!
)

var vestelRegCurrents = []uint16{1008, 1010, 1012} // current readings // todo this are mA! wallbe is A!

// Vestel is an api.ChargeController implementation for Vestel/Hymes wallboxes with Ethernet (SW modells).
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Vestel struct {
	conn *modbus.Connection
	// encoding string
}

func init() {
	registry.Add("vestel", NewVestelFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateVestel -b *Vestel -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.ChargerEx,MaxCurrentMillis,func(current float64) error"

// NewVestelFromConfig creates a Vestel charger from generic config
func NewVestelFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI string
		ID  uint8
	}{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewVestel(cc.URI)
	if err != nil {
		return nil, err
	}

	return wb, err

	// var currentPower func() (float64, error)
	// if cc.Meter.Power {
	// 	currentPower = wb.currentPower
	// }

	// var totalEnergy func() (float64, error)
	// if cc.Meter.Energy {
	// 	totalEnergy = wb.totalEnergy
	// }

	// var currents func() (float64, float64, float64, error)
	// if cc.Meter.Currents {
	// 	currents = wb.currents
	// }

	// var maxCurrentMillis func(float64) error
	// if !cc.Legacy {
	// 	maxCurrentMillis = wb.maxCurrentMillis
	// }

	// // special case for SDM meters
	// if encoding := strings.ToLower(cc.Meter.Encoding); strings.HasPrefix(encoding, encodingSDM) {
	// 	wb.encoding = encodingSDM
	// }

	// return decorateVestel(wb, currentPower, totalEnergy, currents, maxCurrentMillis), nil
}

// NewVestel creates a Vestel charger
func NewVestel(uri string, slaveID uint8) (*Vestel, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.TcpFormat, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("vestel")
	conn.Logger(log.TRACE)

	wb := &Vestel{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Vestel) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[1])), nil
}

// Enabled implements the api.Charger interface
func (wb *Vestel) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(vestelRegActualCurrent, 1)
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

	_, err := wb.conn.WriteSingleCoil(vestelRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Vestel) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	u := uint16(10 * current)
	_, err := wb.conn.WriteSingleRegister(vestelRegActualCurrent, u)

	return err
}

var _ api.ChargeTimer = (*Vestel)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Vestel) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	// 2 words, least significant word first
	secs := uint64(b[3])<<16 | uint64(b[2])<<24 | uint64(b[1]) | uint64(b[0])<<8
	return time.Duration(time.Duration(secs) * time.Second), nil
}

var _ api.Meter = (*Vestel)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Vestel) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*Vestel)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Vestel) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

var _ api.MeterCurrent = (*Vestel)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Vestel) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range vestelRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, float64(binary.BigEndian.Uint32(b))/1e3)
	}

	return currents[0], currents[1], currents[2], nil
}

// Diagnose implements the Diagnosis interface
func (wb *Vestel) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(vestelRegFirmware, 50); err == nil {
		fmt.Printf("Firmware:\t%s\n", encoding.StringSwapped(b))
	}
}
