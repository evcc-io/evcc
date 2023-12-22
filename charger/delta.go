package charger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	//TODO: "github.com/evcc-io/evcc/util/sponsor"
)

// Delta charger implementation
type Delta struct {
	conn   *modbus.Connection
	curr   uint32
	phases uint32
}

const (
	//EV Charger
	deltaRegState  = 0x0064 // 100 - Charger State - Input Register, UINT16 0: not ready, 1: operational, 10: faulted, 255: not responding
	deltaRegCount  = 0x0066 // 102 - EVSE Count - Input Register, UINT16
	deltaRegSerial = 0x006E // 110 - Charger Serial - Input Register, STRING20
	deltaRegModel  = 0x0082 // 130 - Charger Model - Input Register, STRING20

	deltaRegCommunicationTimeoutEnabled = 0x00C9 // 201 - Communication Timeout Enabled - Write Holding, UINT16 0/1
	deltaRegCommunicationTimeout        = 0x00CA // 202 - Communication Timeout - Write Holding, UINT16 [s]
	deltaRegFallbackPower               = 0x00CB // 203 - Fallback Power - Write Holding, UINT32 [W] ==> [mA]???

	//EVSE
	deltaRegEvseState                = 0x03E8 // 1000 - EVSE State - Input Register, UINT16 0: Unavailable, 1: Available, 2: Occupied, 3: Preparing, 4: Charging, 5: Finishing, 6: Suspended EV, 7: Suspended EVSE, 8: Not ready, 9: Faulted
	deltaRegEvseCurrentChargingPower = 0x03ED // 1005 - EVSE Current Charging Power - Input Register, UINT32 [W]
	deltaRegEvseChargingCurrentLimit = 0x0640 // 1600 - EVSE Charging Power Limit - Write Holding, UINT32 [mA]
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

	//TODO:
	//if !sponsor.IsAuthorized() {
	//	return nil, api.ErrSponsorRequired
	//}

	log := util.NewLogger("delta")
	conn.Logger(log.TRACE)

	wb := &Delta{
		conn:   conn,
		curr:   6000, // assume min current
		phases: 2,
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
	b, err := wb.conn.ReadInputRegisters(deltaRegEvseCurrentChargingPower, 2)
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
	//Delta expects Power in Watts. Convert current to Watts considering phases from vehicle
	//TODO: Can we get the number of phases of the currently connected vehicle somehow??
	var power = current / 1000 * 230 * wb.phases

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, power)

	_, err := wb.conn.WriteMultipleRegisters(deltaRegEvseChargingCurrentLimit, 2, b)
	return err
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
	b, err := wb.conn.ReadInputRegisters(deltaRegEvseCurrentChargingPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
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
	if b, err := wb.conn.ReadInputRegisters(deltaRegEvseChargingCurrentLimit, 2); err == nil {
		fmt.Printf("\tCharging current limit:\t%dmA\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegEvseCurrentChargingPower, 2); err == nil {
		fmt.Printf("\tCurrent charging power:\t%dW\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegState, 1); err == nil {
		fmt.Printf("\tState:\t%x\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegEvseState, 1); err == nil {
		fmt.Printf("\tEVSE State:\t%x\n", b)
	}
}
