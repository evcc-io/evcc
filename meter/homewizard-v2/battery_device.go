package v2

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// BatteryMeasurement contains full measurement data from battery device (HWE-BAT)
type BatteryMeasurement struct {
	// Energy measurements
	EnergyImportkWh float64 `json:"energy_import_kwh"`
	EnergyExportkWh float64 `json:"energy_export_kwh"`

	// Power measurements
	PowerW float64 `json:"power_w"`

	// Electrical measurements
	VoltageV    float64 `json:"voltage_v"`
	CurrentA    float64 `json:"current_a"`
	FrequencyHz float64 `json:"frequency_hz"`

	// Battery-specific measurements
	StateOfChargePct float64 `json:"state_of_charge_pct"`
	Cycles           int     `json:"cycles"`
}

// BatteryDevice represents a battery (HWE-BAT) for SoC and power monitoring
type BatteryDevice struct {
	*deviceBase
	measurement *util.Monitor[BatteryMeasurement]
}

// NewBatteryDevice creates a new battery device instance
func NewBatteryDevice(host, token string, timeout time.Duration) *BatteryDevice {
	d := &BatteryDevice{
		deviceBase:  newDeviceBase(DeviceTypeBattery, host, token, timeout),
		measurement: util.NewMonitor[BatteryMeasurement](timeout),
	}

	// Create connection with message handler
	d.conn = NewConnection(host, token, d.handleMessage)

	return d
}

// handleMessage routes incoming WebSocket messages for battery
func (d *BatteryDevice) handleMessage(msgType string, data json.RawMessage) error {
	switch msgType {
	case "measurement":
		var m BatteryMeasurement
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("unmarshal battery measurement: %w", err)
		}
		d.measurement.Set(m)
		d.log.TRACE.Printf("updated battery measurement: soc=%.1f%%, power=%.1fW", m.StateOfChargePct, m.PowerW)

	case "device", "system", "user":
		// Ignore device info, system messages, and user messages
		d.log.TRACE.Printf("ignoring message type: %s", msgType)

	default:
		d.log.TRACE.Printf("unknown message type: %s", msgType)
	}

	return nil
}

// GetMeasurement returns the latest battery measurement data
func (d *BatteryDevice) GetMeasurement() (BatteryMeasurement, error) {
	m, err := d.measurement.Get()
	if err != nil {
		return BatteryMeasurement{}, api.ErrTimeout
	}
	return m, nil
}
