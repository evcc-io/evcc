package charger

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

type SchneiderG4S struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	schneiderG4SRegCpwState            = 6    // 1 RO
	schneiderG4SRegEvState             = 8    // 1 RO
	schneiderG4SRegLastStopCause       = 9    // 1 RO
	schneiderG4SRegCommandStatus       = 20   // 1 RO
	schneiderG4SRegChargingTime        = 30   // 2 RO (seconds)
	schneiderG4SRegSetCommand          = 150  // 1 RW
	schneiderG4SRegSetPoint            = 301  // 1 RW unsigned (A)
	schneiderG4SRegCurrents            = 350  // 6 RO float (A)
	schneiderG4SRegEnergy              = 356  // 2 RO uint32t (Wh)
	schneiderG4SRegPower               = 358  // 2 RO float (kW)
	schneiderG4SRegVoltages            = 366  // 6 RO float (V)
	schneiderG4SRegLifebit             = 932  // 1 RW
	schneiderG4SRegSessionChargingTime = 2004 // 2 RO (seconds)

	schneiderG4SDisabled = uint16(0)
)

func init() {
	registry.Add("schneider-g4s", NewSchneiderG4SFromConfig)
}

func NewSchneiderG4SFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSchneiderG4S(cc.URI, cc.ID)
}

func NewSchneiderG4S(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("schneider-g4s")
	conn.Logger(log.TRACE)

	wb := &SchneiderG4S{
		log:  log,
		conn: conn,
		curr: 6,
	}

	// get initial state from charger
	b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegSetPoint, 1)
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if u := encoding.Uint16(b); u > wb.curr {
		wb.curr = u
	}

	b, err = wb.conn.ReadHoldingRegisters(schneiderG4SRegLifebit, 1)
	if err != nil {
		return nil, fmt.Errorf("heartbeat timeout: %w", err)
	}
	if u := encoding.Uint16(b); u != 2 {
		go wb.heartbeat(2 * time.Second)
	}

	return wb, nil
}

func (wb *SchneiderG4S) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.WriteSingleRegister(schneiderG4SRegLifebit, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *SchneiderG4S) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegEvState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := encoding.Uint16(b)

	switch s {
	case 0:
		return api.StatusA, nil
	case 1, 2:
		return api.StatusB, nil
	case 3, 4:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *SchneiderG4S) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegSetPoint, 1)
	if err != nil {
		return false, err
	}

	return encoding.Uint16(b) != schneiderG4SDisabled, nil
}

// Enable implements the api.Charger interface
func (wb *SchneiderG4S) Enable(enable bool) error {
	u := schneiderG4SDisabled
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(schneiderG4SRegSetPoint, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *SchneiderG4S) MaxCurrent(current int64) error {
	_, err := wb.conn.WriteSingleRegister(schneiderG4SRegSetPoint, uint16(current))
	if err == nil {
		wb.curr = uint16(current)
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (wb *SchneiderG4S) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32LswFirst(b)) * 1e3, nil
}

var _ api.MeterEnergy = (*SchneiderG4S)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SchneiderG4S) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint64LswFirst(b)) / 1e3, nil
}

var _ api.PhaseCurrents = (*SchneiderG4S)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *SchneiderG4S) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(schneiderG4SRegCurrents)
}

var _ api.PhaseVoltages = (*SchneiderG4S)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *SchneiderG4S) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(schneiderG4SRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *SchneiderG4S) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(encoding.Float32LswFirst(b[4*i:]))
	}

	return res[0], res[1], res[2], nil
}

var _ api.ChargeTimer = (*SchneiderG4S)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *SchneiderG4S) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegSessionChargingTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint16(b)) * time.Second, nil
}

var _ api.Diagnosis = (*SchneiderG4S)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SchneiderG4S) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegCpwState, 1); err == nil {
		fmt.Printf("\tcpwState:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegEvState, 1); err == nil {
		fmt.Printf("\tevState:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegLastStopCause, 1); err == nil {
		fmt.Printf("\tLast stop cause:\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegCommandStatus, 1); err == nil {
		fmt.Printf("\tCommand status:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegSetCommand, 1); err == nil {
		fmt.Printf("\tSet command:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegSetPoint, 1); err == nil {
		fmt.Printf("\tSet Point:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegLifebit, 1); err == nil {
		fmt.Printf("\tLifebit:\t\t%d\n", encoding.Uint16(b))
	}
}
