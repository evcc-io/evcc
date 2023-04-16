package charger

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// https://www.keba.com/en/emobility/service-support/downloads/Downloads
// https://www.keba.com/download/x/dea7ae6b84/kecontactp30modbustcp_pgen.pdf

// Keba is an api.Charger implementation
type Keba struct {
	conn *modbus.Connection
}

const (
	kebaRegChargingState = 1000
	kebaRegCableState    = 1004
	kebaRegCurrents      = 1008 // 6 regs, mA
	kebaRegSerial        = 1014 // leading zeros trimmed
	kebaRegProduct       = 1016
	kebaRegFirmware      = 1018
	kebaRegPower         = 1020 // mW
	kebaRegEnergy        = 1036 // Wh
	kebaRegVoltages      = 1040 // 6 regs, V
	kebaRegPhases        = 1215
	kebaRegRfid          = 1500 // hex
	kebaRegSessionEnergy = 1502 // Wh
	kebaRegPhaseSource   = 1550
	kebaRegPhaseState    = 1552
	kebaRegMaxCurrent    = 5004 // mA
	kebaRegEnable        = 5014
	kebaRegSetPhase      = 5052
)

func init() {
	registry.Add("keba-modbus", NewKebaFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateKeba -b *Keba -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewKebaFromConfig creates a new Keba ModbusTCP charger
func NewKebaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewKeba(cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	b, err := wb.conn.ReadHoldingRegisters(kebaRegProduct, 2)
	if err != nil {
		return nil, err
	}

	var currentPower, totalEnergy func() (float64, error)
	var currents, voltages func() (float64, float64, float64, error)
	if features := binary.BigEndian.Uint32(b); features&0xFF00 > 0 {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
		currents = wb.currents
		voltages = wb.voltages
	}

	b, err = wb.conn.ReadHoldingRegisters(kebaRegPhaseSource, 2)
	if err != nil {
		return nil, err
	}

	var phases func(int) error
	if source := binary.BigEndian.Uint32(b); source == 3 {
		phases = wb.phases1p3p
	}

	return decorateKeba(wb, currentPower, totalEnergy, currents, voltages, phases), nil
}

// NewKeba creates a new charger
func NewKeba(uri string, slaveID uint8) (*Keba, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("keba")
	conn.Logger(log.TRACE)

	// per Keba docs
	conn.Delay(500 * time.Millisecond)

	wb := &Keba{
		conn: conn,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Keba) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegChargingState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch status := binary.BigEndian.Uint16(b); status {
	case 0, 1:
		return api.StatusA, nil
	case 2:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", status)
	}
}

// Enabled implements the api.Charger interface
func (wb *Keba) Enabled() (bool, error) {
	return false, nil
}

// Enable implements the api.Charger interface
func (wb *Keba) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(kebaRegEnable, 1, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Keba) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Keba)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Keba) MaxCurrentMillis(current float64) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current*1000))
	_, err := wb.conn.WriteMultipleRegisters(kebaRegMaxCurrent, 1, b)
	return err
}

// currentPower implements the api.Meter interface
func (wb *Keba) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Keba) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *Keba) currents() (float64, float64, float64, error) {
	var res [3]float64
	for i := uint16(0); i < 3; i++ {
		b, err := wb.conn.ReadHoldingRegisters(kebaRegCurrents+2*i, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = float64(binary.BigEndian.Uint32(b)) / 1e3
	}

	return res[0], res[1], res[2], nil
}

// voltages implements the api.PhaseVoltages interface
func (wb *Keba) voltages() (float64, float64, float64, error) {
	var res [3]float64
	for i := uint16(0); i < 3; i++ {
		b, err := wb.conn.ReadHoldingRegisters(kebaRegVoltages+2*i, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = float64(binary.BigEndian.Uint32(b))
	}

	return res[0], res[1], res[2], nil
}

var _ api.Identifier = (*Keba)(nil)

// Identify implements the api.Identifier interface
func (wb *Keba) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegRfid, 2)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Keba) phases1p3p(phases int) error {
	b := make([]byte, 2)
	if phases == 3 {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(kebaRegPhases, 1, b)
	return err
}

var _ api.Diagnosis = (*Keba)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Keba) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegSerial, 2); err == nil {
		fmt.Printf("\tSerial:\t%s\n", strings.TrimLeft(string(b), "0"))
	}
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegFirmware, 2); err == nil {
		fmt.Printf("\tFirmware:\t%d.%d.%d\n", b[0], b[1], b[2])
	}
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegProduct, 2); err == nil {
		fmt.Printf("\tProduct:\t%0x\n", b)
	}
}
