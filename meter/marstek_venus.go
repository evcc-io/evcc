package meter

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Constants for Modbus registers and settings
const (
	// Register addresses
	regControlMode = 42000 // RS485 Control Mode
	regForceMode   = 42010 // Force Charge/Discharge Mode
	regChargePower = 42020 // Forcible Charge Power
	regWorkMode    = 43000 // User Work Mode

	// Control mode values
	controlModeEnabled  = 21930 // RS485 Control Mode enabled
	controlModeDisabled = 21947 // RS485 Control Mode disabled

	// Force mode values
	forceModeStop   = 0 // Stop charging/discharging
	forceModeCharge = 1 // Force charging mode

	// Work mode values
	workModeAntiFeed = 1 // Anti-Feed mode

	// Power settings
	maxChargePowerWatts = 2500 // Maximum charge power in watts
)

// init registers the Marstek Venus meter type
func init() {
	registry.Add("marstek-venus", NewMarstekVenusFromConfig)
}

// MarstekVenus represents the Marstek Venus battery storage system
type MarstekVenus struct {
	conn     *modbus.Connection
	log      *util.Logger
	usage    string
	capacity float64
}

// NewMarstekVenusFromConfig creates a Marstek Venus meter from generic configuration
func NewMarstekVenusFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		ID       uint8
		Timeout  string
		Capacity float64
	}{
		Timeout: "5s", // Default timeout
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %v", err)
	}

	// Parse timeout
	timeoutDuration, err := time.ParseDuration(cc.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %v", err)
	}

	// Create Modbus TCP connection
	ctx := context.Background()
	conn, err := modbus.NewConnection(ctx, cc.URI, "", "", 0, modbus.Tcp, cc.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create modbus connection: %v", err)
	}

	// Set timeout and enable trace logging
	conn.Timeout(timeoutDuration)
	log := util.NewLogger("marstek-venus")
	conn.Logger(log.TRACE)

	m := &MarstekVenus{
		conn:     conn,
		log:      log,
		usage:    "battery", // Hardcoded as battery meter
		capacity: cc.Capacity,
	}

	// Probe registers to confirm availability
	if _, err := m.conn.ReadHoldingRegisters(32104, 1); err != nil {
		m.log.WARN.Printf("SOC register 32104 not available: %v", err)
	}

	return m, nil
}

// CurrentPower implements the api.Meter interface
func (m *MarstekVenus) CurrentPower() (float64, error) {
	b, err := m.conn.ReadHoldingRegisters(32202, 2) // AC Power (32-bit, Watt)
	if err != nil {
		return 0, fmt.Errorf("failed to read power: %v", err)
	}
	power := int32(binary.BigEndian.Uint32(b)) // Assumes BigEndian
	return float64(power), nil
}

// Soc implements the api.Battery interface
func (m *MarstekVenus) Soc() (float64, error) {
	b, err := m.conn.ReadHoldingRegisters(32104, 1) // Battery SOC (16-bit, %)
	if err != nil {
		return 0, fmt.Errorf("failed to read soc: %v", err)
	}
	soc := binary.BigEndian.Uint16(b) // Assumes BigEndian
	if soc > 100 {
		return 0, api.ErrNotAvailable
	}
	return float64(soc), nil
}

// SetBatteryMode implements the api.BatteryController interface
func (m *MarstekVenus) SetBatteryMode(mode api.BatteryMode) error {
	m.log.INFO.Printf("Setting battery mode to (mode=%v)", mode)
	switch mode {
	case api.BatteryNormal:
		// Step 1: Set RS485 Control Mode = Enabled (21930)
		if _, err := m.conn.WriteSingleRegister(regControlMode, controlModeEnabled); err != nil {
			return fmt.Errorf("failed to enable RS485 control mode: %v", err)
		}
		// Step 2: Set User Work Mode = Anti-Feed (1)
		if _, err := m.conn.WriteSingleRegister(regWorkMode, workModeAntiFeed); err != nil {
			return fmt.Errorf("failed to set anti-feed mode: %v", err)
		}
		// Step 3: Set RS485 Control Mode = Disabled (21947)
		if _, err := m.conn.WriteSingleRegister(regControlMode, controlModeDisabled); err != nil {
			return fmt.Errorf("failed to disable RS485 control mode: %v", err)
		}
		m.log.INFO.Printf("Battery mode set to Normal/Anti-Feed (mode=%v)", mode)
	case api.BatteryCharge:
		// Step 1: Set RS485 Control Mode = Enabled (21930)
		if _, err := m.conn.WriteSingleRegister(regControlMode, controlModeEnabled); err != nil {
			return fmt.Errorf("failed to enable RS485 control mode: %v", err)
		}
		// Step 2: Set Force Charge/Discharge Mode = Charge (1)
		if _, err := m.conn.WriteSingleRegister(regForceMode, forceModeCharge); err != nil {
			return fmt.Errorf("failed to set charge mode: %v", err)
		}
		// Step 3: Set Forcible Charge Power = 2500 watts
		if _, err := m.conn.WriteSingleRegister(regChargePower, maxChargePowerWatts); err != nil {
			return fmt.Errorf("failed to set charge power: %v", err)
		}
		m.log.INFO.Printf("Battery mode set to Charge with %d watts (mode=%v)", maxChargePowerWatts, mode)

	case api.BatteryHold:
		// Step 1: Set RS485 Control Mode = Enabled (21930)
		// Step 1: Set RS485 Control Mode = Enabled (21930)
		if _, err := m.conn.WriteSingleRegister(regControlMode, controlModeEnabled); err != nil {
			return fmt.Errorf("failed to enable RS485 control mode: %v", err)
		}
		// Step 2: Set Force Charge/Discharge = Stop (0)
		if _, err := m.conn.WriteSingleRegister(regForceMode, forceModeStop); err != nil {
			return fmt.Errorf("failed to set stop mode: %v", err)
		}
		m.log.INFO.Printf("Battery mode set to Hold (mode=%v)", mode)

	default:
		return fmt.Errorf("unsupported battery mode: %v", mode)
	}

	return nil
}

// Diagnose implements the api.Diagnosis interface
func (m *MarstekVenus) Diagnose() {
	if b, err := m.conn.ReadHoldingRegisters(32202, 2); err == nil {
		m.log.INFO.Printf("\tPower:\t%d W", int32(binary.BigEndian.Uint32(b)))
	}
	if b, err := m.conn.ReadHoldingRegisters(32104, 1); err == nil {
		m.log.INFO.Printf("\tSOC:\t%d%%", binary.BigEndian.Uint16(b))
	}
	if b, err := m.conn.ReadHoldingRegisters(regControlMode, 1); err == nil {
		m.log.INFO.Printf("\tControl Mode:\t%d", binary.BigEndian.Uint16(b))
	}
	if b, err := m.conn.ReadHoldingRegisters(regForceMode, 1); err == nil {
		m.log.INFO.Printf("\tForce Mode:\t%d", binary.BigEndian.Uint16(b))
	}
	if b, err := m.conn.ReadHoldingRegisters(regChargePower, 1); err == nil {
		m.log.INFO.Printf("\tCharge Power:\t%d W", binary.BigEndian.Uint16(b))
	}
	if b, err := m.conn.ReadHoldingRegisters(regWorkMode, 1); err == nil {
		m.log.INFO.Printf("\tWork Mode:\t%d", binary.BigEndian.Uint16(b))
	}
}
