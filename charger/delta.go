package charger

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// Delta charger implementation
type Delta struct {
	log     *util.Logger
	conn    *modbus.Connection
	lp      loadpoint.API
	mu      sync.Mutex
	curr    float64
	base    uint16
	enabled bool
}

const (
	// EV Charger
	// Read Input Registers (0x04)
	deltaRegState   = 100 // Charger State - UINT16 0: not ready, 1: operational, 10: faulted, 255: not responding
	deltaRegVersion = 101 // Charger Version - UINT16
	deltaRegCount   = 102 // Charger EVSE Count - UINT16
	deltaRegError   = 103 // Charger Error - UINT16
	deltaRegSerial  = 110 // Charger Serial - STRING20
	deltaRegModel   = 130 // Charger Model - STRING20

	// Write Multiple Registers (0x10)
	deltaRegCommunicationTimeoutEnabled = 201 // Communication Timeout Enabled 0/1
	deltaRegCommunicationTimeout        = 202 // Communication Timeout [s]
	deltaRegFallbackPower               = 203 // Fallback Power [W]

	// EVSE - The following Register tables are defined as repeating blocks for each single EVSE
	// Read Input Registers (0x04)
	deltaRegEvseState                 = 0   // EVSE State - UINT16 0: Unavailable, 1: Available, 2: Occupied, 3: Preparing, 4: Charging, 5: Finishing, 6: Suspended EV, 7: Suspended EVSE, 8: Not ready, 9: Faulted
	deltaRegEvseChargerState          = 1   // EVSE Charger State - UINT16 0: Charging process not started (no vehicle connected), 1: Connected, waiting for release (by RFID or local), 2: Charging process starts, 3: Charging, 4: Suspended (paused), 5: Charging process successfully completed (vehicle still plugged in), 6: Charging process completed by user (vehicle still plugged in), 7: Charging ended with error (vehicle still connected)
	deltaRegEvseActualOutputVoltage   = 3   // EVSE Actual Output Voltage - FLOAT32 [V]
	deltaRegEvseActualChargingPower   = 5   // EVSE Actual Charging Power - UINT32 [W]
	deltaRegEvseActualChargingCurrent = 7   // EVSE Actual Charging Current - FLOAT32 [A]
	deltaRegEvseActualOutputPower     = 9   // EVSE Actual Output Power - FLOAT32 [W]
	deltaRegEvseSoc                   = 11  // EVSE SOC [%/10]
	deltaRegEvseChargingTime          = 17  // EVSE Charging Time [s]
	deltaRegEvseChargedEnergy         = 19  // EVSE Charged Energy [Wh]
	deltaRegEvseRfidUID               = 100 // EVSE Used Authentication ID - STRING

	// Write Multiple Registers (0x10)
	deltaRegEvseChargingPowerLimit = 600 // EVSE Charging Power Limit - UINT32 [W]
	deltaRegEvseSuspendCharging    = 602 // EVSE Suspend Charging - UINT16 - 0: no pause, 1 charging pause (lock on)
)

func init() {
	registry.Add("delta", NewDeltaFromConfig)
}

// NewDeltaFromConfig creates a Delta charger from generic config
func NewDeltaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Connector       uint16
		modbus.Settings `mapstructure:",squash"`
	}{
		Connector: 1,
		Settings: modbus.Settings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDelta(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID, cc.Connector)
}

// NewDelta creates Delta charger
func NewDelta(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, connector uint16) (api.Charger, error) {
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
		log:  log,
		conn: conn,
		curr: 6000, // assume min current
	}

	wb.base = connector * 1000

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(deltaRegCommunicationTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := encoding.Uint16(b); u > 0 {
		go wb.heartbeat(time.Duration(u) * time.Second / 2)
	}

	return wb, err
}

func (wb *Delta) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		wb.mu.Lock()
		var curr float64
		if wb.enabled {
			curr = wb.curr
		}
		wb.mu.Unlock()
		if err := wb.setCurrent(curr); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Delta) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+deltaRegEvseChargerState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	// 0: Charging process not started (no vehicle connected)
	// 1: Connected, waiting for release (by RFID or local)
	// 2: Charging process starts
	// 3: Charging
	// 4: Suspended (loading paused)
	// 5: Charging process successfully completed (vehicle still plugged in)
	// 6: Charging process completed by user (vehicle still plugged in)
	// 7: Charging ended with error (vehicle still connected)
	switch s := encoding.Uint16(b); s {
	case 0:
		return api.StatusA, nil
	case 1, 2, 4, 5, 6, 7:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Delta) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+deltaRegEvseChargingPowerLimit, 2)
	if err != nil {
		return false, err
	}

	return encoding.Uint32(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Delta) Enable(enable bool) error {
	var curr float64
	if enable {
		wb.mu.Lock()
		curr = wb.curr
		wb.mu.Unlock()
	}

	err := wb.setCurrent(curr)
	if err == nil {
		wb.mu.Lock()
		wb.enabled = enable
		wb.mu.Unlock()
	}

	return err
}

// setCurrent writes the current limit in A
func (wb *Delta) setCurrent(current float64) error {
	activePhases := 3
	if wb.lp != nil {
		activePhases = wb.lp.ActivePhases()
	}

	b := make([]byte, 4)
	encoding.PutUint32(b, uint32(math.Trunc(230.0*current*float64(activePhases))))

	_, err := wb.conn.WriteMultipleRegisters(wb.base+deltaRegEvseChargingPowerLimit, 2, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Delta) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Delta)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Delta) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	err := wb.setCurrent(current)
	if err == nil {
		wb.mu.Lock()
		wb.curr = current
		wb.mu.Unlock()
	}

	return err
}

var _ api.Meter = (*Delta)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Delta) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+deltaRegEvseActualChargingPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(b)), err
}

var _ api.ChargeRater = (*Delta)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Delta) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+deltaRegEvseChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(b)) / 1e3, err
}

var _ api.Identifier = (*Delta)(nil)

// Identify implements the api.Identifier interface
func (wb *Delta) Identify() (string, error) {
	b, err := wb.conn.ReadInputRegisters(wb.base+deltaRegEvseRfidUID, 6)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

var _ api.Diagnosis = (*Delta)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Delta) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(deltaRegState, 1); err == nil {
		fmt.Printf("\tState:\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegVersion, 1); err == nil {
		fmt.Printf("\tVersion:\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegCount, 1); err == nil {
		fmt.Printf("\tEVSE Count:\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegError, 1); err == nil {
		fmt.Printf("\tError:\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegSerial, 20); err == nil {
		fmt.Printf("\tSerial:\t%s\n", bytesAsString(b))
	}
	if b, err := wb.conn.ReadInputRegisters(deltaRegModel, 20); err == nil {
		fmt.Printf("\tModel:\t%s\n", bytesAsString(b))
	}
}

var _ loadpoint.Controller = (*Delta)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *Delta) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
