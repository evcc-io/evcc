package charger

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/spf13/cast"
	"github.com/volkszaehler/mbmd/encoding"
)

// https://www.nrgkick.com/wp-content/uploads/2024/07/local_api_docu_simulate.html

// NRGKickGen2 charger implementation
type NRGKickGen2 struct {
	conn *modbus.Connection
}

const (
	// All register use LittleEndian
	// Read only (0x03)
	nrgKickGen2Serial            = 0  // 11 regs
	nrgKickGen2ModelType         = 11 // 16 regs
	nrgKickGen2MaxPhases         = 36
	nrgKickGen2SoftwareVersionSM = 122 // 8 regs
	// Read (0x03) / Write (0x06, 0x16) Registers
	nrgKickGen2ChargingCurrent = 194 // A, factor 10
	nrgKickGen2Enabled         = 195
	nrgKickGen2Phases          = 198
	// Read only (0x03)
	nrgKickGen2TotalChargedEnergy = 199 // Wh, 4 regs
	nrgKickGen2ChargedEnergy      = 203 // Wh, 2 regs
	nrgKickGen2TotalActivePower   = 210 // W, 2 regs, factor 1000
	nrgKickGen2PhaseVoltages      = 217 // factor 100
	nrgKickGen2PhaseCurrents      = 220 // factor 1000
	nrgKickGen2RegStatus          = 251
	nrgKickGen2RegRelais          = 253
	nrgKickGen2RegRCD             = 255
	nrgKickGen2RegWarning         = 256
	nrgKickGen2RegError           = 257
)

func init() {
	registry.Add("nrggen2", NewNRGKickGen2FromConfig)
}

//go:generate decorate -f decorateNRGKickGen2 -b *NRGKickGen2 -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewNRGKickGen2FromConfig creates a NRGKickGen2 charger from generic config
func NewNRGKickGen2FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Phases1p3p         bool
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1, // default
		},
		Phases1p3p: false,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	nrg, err := NewNRGKickGen2(cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	var phasesS func(int) error
	if cc.Phases1p3p {
		// user could have an adapter plug which doesn't support 3 phases
		if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2MaxPhases, 1); err == nil {
			if maxPhases := encoding.Uint16(b); maxPhases > 1 {
				phasesS = nrg.phases1p3p
			}
		}
	}

	return decorateNRGKickGen2(nrg, phasesS), nil
}

// NewNRGKickGen2 creates NRGKickGen2 charger
func NewNRGKickGen2(uri string, slaveID uint8) (*NRGKickGen2, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("nrggen2")
	conn.Logger(log.TRACE)

	nrg := &NRGKickGen2{
		conn: conn,
	}

	return nrg, nil
}

// Status implements the api.Charger interface
func (nrg *NRGKickGen2) Status() (api.ChargeStatus, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2RegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	// 0 - "UNKNOWN",
	// 1 - "STANDBY",
	// 2 - "CONNECTED",
	// 3 - "CHARGING",
	// 6 - "ERROR",
	// 7 - "WAKEUP"
	switch status := binary.BigEndian.Uint16(b); status {
	case 0:
		return api.StatusNone, nil
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 6:
		// 0 - "NO_ERROR",
		// 1 - "GENERAL_ERROR",
		// 2 - "32A_ATTACHMENT_ON_16A_UNIT",
		// 3 - "VOLTAGE_DROP_DETECTED",
		// 4 - "UNPLUG_DETECTION_TRIGGERED",
		// 5 - "TYPE2_NOT_AUTHORIZED",
		// 16 - "RESIDUAL_CURRENT_DETECTED",
		// 32 - "CP_SIGNAL_VOLTAGE_ERROR",
		// 33 - "CP_SIGNAL_IMPERMISSIBLE",
		// 34 - "EV_DIODE_FAULT",
		// 48 - "PE_SELF_TEST_FAILED",
		// 49 - "RCD_SELF_TEST_FAILED",
		// 50 - "RELAY_SELF_TEST_FAILED",
		// 51 - "PE_AND_RCD_SELF_TEST_FAILED",
		// 52 - "PE_AND_RELAY_SELF_TEST_FAILED",
		// 53 - "RCD_AND_RELAY_SELF_TEST_FAILED",
		// 54 - "PE_AND_RCD_AND_RELAY_SELF_TEST_FAILED",
		// 64 - "SUPPLY_VOLTAGE_ERROR",
		// 65 - "PHASE_SHIFT_ERROR",
		// 66 - "OVERVOLTAGE_DETECTED",
		// 67 - "UNDERVOLTAGE_DETECTED",
		// 68 - "OVERVOLTAGE_WITHOUT_PE_DETECTED",
		// 69 - "UNDERVOLTAGE_WITHOUT_PE_DETECTED",
		// 70 - "UNDERFREQUENCY_DETECTED",
		// 71 - "OVERFREQUENCY_DETECTED",
		// 72 - "UNKNOWN_FREQUENCY_TYPE",
		// 73 - "UNKNOWN_GRID_TYPE",
		// 80 - "GENERAL_OVERTEMPERATURE",
		// 81 - "HOUSING_OVERTEMPERATURE",
		// 82 - "ATTACHMENT_OVERTEMPERATURE",
		// 83 - "DOMESTIC_PLUG_OVERTEMPERATURE",
		// x - "UNKNOWN"
		b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2RegError, 1)
		if err != nil {
			return api.StatusNone, err
		}
		return api.StatusF, fmt.Errorf("%d", binary.BigEndian.Uint16(b))
	case 7:
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("unhandled status type")
	}
}

// Enabled implements the api.Charger interface
func (nrg *NRGKickGen2) Enabled() (bool, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2Enabled, 1)
	if err != nil {
		return false, err
	}

	// 0 = no charge pause, 1 = charge pause
	return binary.BigEndian.Uint16(b) == 0, nil
}

// Enable implements the api.Charger interface
func (nrg *NRGKickGen2) Enable(enable bool) error {
	_, err := nrg.conn.WriteSingleRegister(nrgKickGen2Enabled, cast.ToUint16(!enable))
	return err
}

// MaxCurrent implements the api.Charger interface
func (nrg *NRGKickGen2) MaxCurrent(current int64) error {
	return nrg.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*NRGKickGen2)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (nrg *NRGKickGen2) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	_, err := nrg.conn.WriteSingleRegister(nrgKickGen2ChargingCurrent, uint16(math.Trunc(current*10)))
	return err
}

func (nrg *NRGKickGen2) GetMaxCurrent() (float64, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2ChargingCurrent, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, nil
}

var _ api.Meter = (*NRGKickGen2)(nil)

// CurrentPower implements the api.Meter interface
func (nrg *NRGKickGen2) CurrentPower() (float64, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2TotalActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int32LswFirst(b)) * 1e-3, nil
}

var _ api.MeterEnergy = (*NRGKickGen2)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (nrg *NRGKickGen2) TotalEnergy() (float64, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2TotalChargedEnergy, 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint64LswFirst(b)) * 1e-3, nil
}

var _ api.PhaseCurrents = (*NRGKickGen2)(nil)

// Currents implements the api.PhaseCurrents interface
func (nrg *NRGKickGen2) Currents() (float64, float64, float64, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2PhaseCurrents, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) * 1e-3
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseVoltages = (*NRGKickGen2)(nil)

// Currents implements the api.PhaseVoltages interface
func (nrg *NRGKickGen2) Voltages() (float64, float64, float64, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2PhaseVoltages, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) * 1e-2
	}

	return res[0], res[1], res[2], nil
}

var _ api.ChargeRater = (*NRGKickGen2)(nil)

func (nrg *NRGKickGen2) ChargedEnergy() (float64, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2ChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32LswFirst(b)) * 1e-3, nil
}

// Phases1p3p implements the api.PhaseSwitcher interface
func (nrg *NRGKickGen2) phases1p3p(phases int) error {
	// this can return an error, if phase switching isn't activated via the App
	_, err := nrg.conn.WriteSingleRegister(nrgKickGen2Phases, uint16(phases))
	return err
}

var _ api.PhaseGetter = (*NRGKickGen2)(nil)

func (nrg *NRGKickGen2) GetPhases() (int, error) {
	b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2Phases, 1)
	if err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint16(b)), nil
}

var _ api.Diagnosis = (*NRGKickGen2)(nil)

// Diagnose implements the api.Diagnosis interface
func (nrg *NRGKickGen2) Diagnose() {
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2Serial, 11); err == nil {
		fmt.Printf("\tSerial:\t%s\n", bytesAsString(b))
	}
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2ModelType, 16); err == nil {
		fmt.Printf("\tModel:\t%s\n", bytesAsString(b))
	}
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2SoftwareVersionSM, 8); err == nil {
		fmt.Printf("\tSmartModule Version:\t%s\n", bytesAsString(b))
	}
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2RegStatus, 1); err == nil {
		fmt.Printf("\tStatus:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2RegRelais, 1); err == nil {
		fmt.Printf("\tRelais Switching:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2RegRCD, 1); err == nil {
		fmt.Printf("\tRCD:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2RegWarning, 1); err == nil {
		fmt.Printf("\tWarning:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := nrg.conn.ReadHoldingRegisters(nrgKickGen2RegError, 1); err == nil {
		fmt.Printf("\tError:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
