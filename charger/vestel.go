package charger

// todo vestel hymes

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	vestelRegSerial          = 100 // 25
	vestelRegBrand           = 190 // 10
	vestelRegModel           = 210 // 5
	vestelRegFirmware        = 230 // 50
	vestelRegChargeStatus    = 1001
	vestelRegCableStatus     = 1004
	vestelRegChargeTime      = 1508
	vestelRegMaxCurrent      = 5004
	vestelRegPower           = 1020
	vestelRegEnergy          = 1502
	vestelRegFailsafeTimeout = 2002
	vestelRegAlive           = 6000
)

var vestelRegCurrents = []uint16{1008, 1010, 1012}

// Vestel is an api.ChargeController implementation for Vestel/Hymes wallboxes with Ethernet (SW modells).
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Vestel struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
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

	return NewVestel(cc.URI, cc.ID)
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
		log:     log,
		conn:    conn,
		current: 6,
	}

	// 5min failsafe timeout
	if _, err := wb.conn.WriteSingleRegister(vestelRegFailsafeTimeout, 5*60); err != nil {
		return nil, fmt.Errorf("could not set failsafe timeout: %v", err)
	}

	go wb.heartbeat()

	return wb, nil
}

// heartbeat implements the api.ChargerEx interface
func (wb *Vestel) heartbeat() {
	for range time.NewTicker(time.Minute).C {
		if _, err := wb.conn.WriteSingleRegister(vestelRegAlive, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Vestel) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegCableStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	res := api.StatusA
	if binary.BigEndian.Uint16(b) > 0 {
		res = api.StatusB

		b, err := wb.conn.ReadInputRegisters(vestelRegChargeStatus, 1)
		if err != nil {
			return api.StatusNone, err
		}

		if binary.BigEndian.Uint16(b) == 1 {
			res = api.StatusC
		}
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *Vestel) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(vestelRegMaxCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Vestel) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.current
	}

	_, err := wb.conn.WriteSingleRegister(vestelRegMaxCurrent, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Vestel) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(10 * current)
	_, err := wb.conn.WriteSingleRegister(vestelRegMaxCurrent, u)
	if err == nil {
		wb.current = u
	}

	return err
}

var _ api.ChargeTimer = (*Vestel)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Vestel) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	secs := binary.BigEndian.Uint32(b)
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
	if b, err := wb.conn.ReadInputRegisters(vestelRegBrand, 10); err == nil {
		fmt.Printf("Brand:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegModel, 5); err == nil {
		fmt.Printf("Model:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegSerial, 25); err == nil {
		fmt.Printf("Serial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegFirmware, 50); err == nil {
		fmt.Printf("Firmware:\t%s\n", b)
	}
}
