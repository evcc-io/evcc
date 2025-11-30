package v2

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// KWHMeasurement contains power and energy data from kWh meter (HWE-KWH1, HWE-KWH3)
// kWh meters report total energy without tariff breakdown
type KWHMeasurement struct {
	// Energy measurements (total, no tariff breakdown)
	EnergyImportkWh float64 `json:"energy_import_kwh"`
	EnergyExportkWh float64 `json:"energy_export_kwh"`

	// Power measurements
	PowerW   float64 `json:"power_w"`
	PowerL1W float64 `json:"power_l1_w"` // 3-phase only
	PowerL2W float64 `json:"power_l2_w"` // 3-phase only
	PowerL3W float64 `json:"power_l3_w"` // 3-phase only

	// Voltage measurements
	VoltageV   float64 `json:"voltage_v"`    // 1-phase
	VoltageL1V float64 `json:"voltage_l1_v"` // 3-phase
	VoltageL2V float64 `json:"voltage_l2_v"` // 3-phase
	VoltageL3V float64 `json:"voltage_l3_v"` // 3-phase

	// Current measurements
	CurrentA   float64 `json:"current_a"`
	CurrentL1A float64 `json:"current_l1_a"` // 3-phase
	CurrentL2A float64 `json:"current_l2_a"` // 3-phase
	CurrentL3A float64 `json:"current_l3_a"` // 3-phase
}

// KWHDevice represents a kWh meter (HWE-KWH1, HWE-KWH3) for PV monitoring
type KWHDevice struct {
	*deviceBase
	measurement *util.Monitor[KWHMeasurement]
}

// NewKWHDevice creates a new kWh meter device instance
func NewKWHDevice(host, token string, timeout time.Duration) *KWHDevice {
	d := &KWHDevice{
		deviceBase:  newDeviceBase(DeviceTypeKWHMeter, host, token, timeout),
		measurement: util.NewMonitor[KWHMeasurement](timeout),
	}

	// Create connection with message handler
	d.conn = NewConnection(host, token, d.handleMessage)

	return d
}

// handleMessage routes incoming WebSocket messages for kWh meter
func (d *KWHDevice) handleMessage(msgType string, data json.RawMessage) error {
	switch msgType {
	case "measurement":
		var m KWHMeasurement
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("unmarshal kWh measurement: %w", err)
		}
		d.measurement.Set(m)
		d.log.TRACE.Printf("updated kWh measurement: power=%.1fW", m.PowerW)

	case "device", "system":
		// Ignore device info and system messages
		d.log.TRACE.Printf("ignoring message type: %s", msgType)

	default:
		d.log.TRACE.Printf("unknown message type: %s", msgType)
	}

	return nil
}

// GetMeasurement returns the latest kWh meter measurement data
func (d *KWHDevice) GetMeasurement() (KWHMeasurement, error) {
	m, err := d.measurement.Get()
	if err != nil {
		return KWHMeasurement{}, api.ErrTimeout
	}
	return m, nil
}
