package charger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Delta charger implementation
type Delta struct {
	conn *modbus.Connection
	curr uint32
}

const (
	//EV Charger Read Input Registers (0x04)
	deltaRegState  = 100 // Charger State - UINT16 0: not ready, 1: operational, 10: faulted, 255: not responding
	deltaRegCount  = 102 // EVSE Count - UINT16
	deltaRegSerial = 110 // Charger Serial - STRING20
	deltaRegModel  = 130 // Charger Model - STRING20

	//EV Charger Write Multiple Registers (0x10)
	deltaRegCommunicationTimeoutEnabled = 201 // Communication Timeout Enabled 0/1
	deltaRegCommunicationTimeout        = 202 // Communication Timeout [s]
	deltaRegFallbackPower               = 203 // Fallback Power [W]

	//EVSE Read Input Registers (0x04)
	deltaRegEvseState                 = 1000 // EVSE State - UINT16 0: Unavailable, 1: Available, 2: Occupied, 3: Preparing, 4: Charging, 5: Finishing, 6: Suspended EV, 7: Suspended EVSE, 8: Not ready, 9: Faulted
	deltaRegEvseChargerState          = 1001 // EVSE Charger State - 0: Charging process not started (no, vehicle connected), 1: Connected, waiting for release (by, RFID or local), 2: Charging process starts, 3: Charging, 4: Suspended (loading paused), 5: Charging process successfully com-, pleted (vehicle still plugged in), 6: Charging process completed by, user (vehicle still plugged in), 7: Charging ended with error (vehicle, still connected)
	deltaRegEvseActualOutputVoltage   = 1003 // EVSE Actual Output Voltage [V]
	deltaRegEvseActualChargingPower   = 1005 // EVSE Actual Charging Power [W]
	deltaRegEvseActualChargingCurrent = 1005 // EVSE Actual Charging Current [A]

	deltaRegEvseChargingTime  = 1017 // EVSE Charging Time [s]
	deltaRegEvseChargedEnergy = 1019 // EVSE Charged Energy [Wh]

	deltaRegEvseCurrentPowerConsumptionL1 = 1049 // EVSE Current Power Consumption L1 (grid) [W]
	deltaRegEvseCurrentPowerConsumptionL2 = 1051 // EVSE Current Power Consumption L2 (grid) [W]
	deltaRegEvseCurrentPowerConsumptionL3 = 1053 // EVSE Current Power Consumption L3 (grid) [W]

	//EVSE Write Multiple Registers (0x10)
	deltaRegEvseChargingPowerLimit = 1600 // EVSE Charging Power Limit - UINT32 [W]
	deltaRegEvseSuspendCharging    = 1602 // EVSE Suspend Charging - UINT16 - 0: no pause, 1 charging pause (lock on)
)

func init() {
	registry.Add("delta", NewDeltaFromConfig)
}

// NewDeltaFromConfig creates a Delta charger from generic config
func NewDeltaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDelta(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewDelta creates Delta charger
func NewDelta(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("delta")
	conn.Logger(log.TRACE)

	wb := &Delta{
		conn: conn,
		curr: 6000, // assume min current
	}

	// keep-alive
	go func() {
		for range time.Tick(30 * time.Second) {
			_, _ = wb.status()
		}
	}()

	return wb, err
}

func (wb *Delta) status() (byte, error) {
	b, err := wb.conn.ReadInputRegisters(deltaRegEvseState, 1)
	if err != nil {
		return 0, err
	}

	return b[1], nil
}

// Status implements the api.Charger interface
func (wb *Delta) Status() (api.ChargeStatus, error) {
	s, err := wb.status()
	if err != nil {
		return api.StatusNone, err
	}

	switch s {
	case 0: //Unavailable
		return api.StatusNone, nil
	case 1: //Available
		return api.StatusA, nil // State A: Idle
	case 2: //Occupied
		return api.StatusB, nil // State B1: EV Plug in, pending authorization
	case 3: //Preparing
		return api.StatusB, nil // State B2: EV Plug in, EVSE ready for charging (PWM)
	case 4: //Charging
		return api.StatusC, nil // State C2: Charging Contact closed, energy delivering
	case 5: //Finishing
		return api.StatusC, nil
	case 6: //Suspended EV
		return api.StatusE, nil // State E: Error EV / Cable
	case 7: //Suspended EVSE
		return api.StatusF, nil // State F: Error EVSE or simulated disconnect
	case 8: //Not ready
		return api.StatusNone, nil
	case 9: //Faulted
		return api.StatusB, nil
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %0x", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Delta) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(deltaRegEvseActualChargingPower, 2)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint32(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Delta) Enable(enable bool) error {
	var current uint32
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// setCurrent writes the current limit in mA
func (wb *Delta) setCurrent(current uint32) error {
	//Delta expects Power in Watts. Convert current to Watts considering active phases
	vehiclePhases, err := wb.detectActivePhases()
	if err != nil {
		return err
	}
	var power = current / 1000 * 230 * vehiclePhases

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, power)

	_, err := wb.conn.WriteMultipleRegisters(deltaRegEvseChargingPowerLimit, 2, b)
	return err
}

func (wb *Delta) detectActivePhases() (uint32, error) {
	var vehiclePhases uint32

	b, err := wb.conn.ReadInputRegisters(deltaRegEvseCurrentPowerConsumptionL1, 2)
	if err != nil {
		return 0, err
	}

	if binary.BigEndian.Uint32(b) > 0 {
		vehiclePhases++
	}

	b, err = wb.conn.ReadInputRegisters(deltaRegEvseCurrentPowerConsumptionL2, 2)
	if err != nil {
		return 0, err
	}

	if binary.BigEndian.Uint32(b) > 0 {
		vehiclePhases++
	}

	b, err = wb.conn.ReadInputRegisters(deltaRegEvseCurrentPowerConsumptionL3, 2)
	if err != nil {
		return 0, err
	}

	if binary.BigEndian.Uint32(b) > 0 {
		vehiclePhases++
	}

	//Charging not started? Use 3 as fallback
	if vehiclePhases == 0 {
		vehiclePhases = 3
	}

	return vehiclePhases, nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Delta) MaxCurrent(current int64) error {
	return wb.setCurrent(uint32(current))
}

var _ api.ChargerEx = (*Delta)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Delta) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	wb.curr = uint32(current * 1e3)

	return wb.setCurrent(wb.curr)
}

var _ api.Meter = (*Delta)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Delta) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(deltaRegEvseActualChargingPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.ChargeRater = (*Delta)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Delta) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(deltaRegEvseChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b) / 1000), err
}

var _ api.Diagnosis = (*Delta)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Delta) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(deltaRegSerial, 20); err == nil {
		fmt.Printf("\tSerial:\t%x\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegModel, 20); err == nil {
		fmt.Printf("\tModel:\t%x\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegEvseChargingPowerLimit, 2); err == nil {
		fmt.Printf("\tCharging power limit:\t%dmA\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegEvseActualChargingPower, 2); err == nil {
		fmt.Printf("\tCurrent charging power:\t%dW\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegState, 1); err == nil {
		fmt.Printf("\tState:\t%x\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegEvseState, 1); err == nil {
		fmt.Printf("\tEVSE State:\t%x\n", b)
	}
}
