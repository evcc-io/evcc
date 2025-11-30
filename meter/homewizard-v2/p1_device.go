package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// P1Measurement contains power and energy data from P1 meter (HWE-P1)
// P1 meters report energy with tariff breakdown (T1/T2)
type P1Measurement struct {
	// Power measurements
	PowerW   float64 `json:"power_w"`
	PowerL1W float64 `json:"power_l1_w"`
	PowerL2W float64 `json:"power_l2_w"`
	PowerL3W float64 `json:"power_l3_w"`

	// Voltage measurements
	VoltageL1V float64 `json:"voltage_l1_v"`
	VoltageL2V float64 `json:"voltage_l2_v"`
	VoltageL3V float64 `json:"voltage_l3_v"`

	// Current measurements
	CurrentL1A float64 `json:"current_l1_a"`
	CurrentL2A float64 `json:"current_l2_a"`
	CurrentL3A float64 `json:"current_l3_a"`

	// Energy measurements with tariff breakdown
	EnergyImportT1kWh float64 `json:"energy_import_t1_kwh"`
	EnergyImportT2kWh float64 `json:"energy_import_t2_kwh"`
	EnergyExportT1kWh float64 `json:"energy_export_t1_kwh"`
	EnergyExportT2kWh float64 `json:"energy_export_t2_kwh"`
}

// P1Device represents a P1 meter (HWE-P1) for grid monitoring and battery control
type P1Device struct {
	*deviceBase
	measurement   *util.Monitor[P1Measurement]
	batteriesData *util.Monitor[BatteriesData]
}

// NewP1Device creates a new P1 meter device instance
func NewP1Device(host, token string, timeout time.Duration) *P1Device {
	d := &P1Device{
		deviceBase:    newDeviceBase(DeviceTypeP1Meter, host, token, timeout),
		measurement:   util.NewMonitor[P1Measurement](timeout),
		batteriesData: util.NewMonitor[BatteriesData](timeout),
	}

	// Create connection with message handler, subscribe to measurement and batteries topics
	d.conn = NewConnection(host, token, d.handleMessage, "measurement", "batteries")

	return d
}

// handleMessage routes incoming WebSocket messages for P1 meter
func (d *P1Device) handleMessage(msgType string, data json.RawMessage) error {
	switch msgType {
	case "measurement":
		var m P1Measurement
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("unmarshal P1 measurement: %w", err)
		}
		d.measurement.Set(m)
		d.log.TRACE.Printf("updated P1 measurement: power=%.1fW", m.PowerW)

	case "batteries":
		var b BatteriesData
		if err := json.Unmarshal(data, &b); err != nil {
			return fmt.Errorf("unmarshal batteries data: %w", err)
		}
		d.batteriesData.Set(b)
		d.log.TRACE.Printf("updated batteries data: mode=%s, charge_limit=%.0fW, discharge_limit=%.0fW",
			b.Mode, b.MaxConsumptionW, b.MaxProductionW)

	case "device", "system":
		// Ignore device info and system messages
		d.log.TRACE.Printf("ignoring message type: %s", msgType)

	default:
		d.log.TRACE.Printf("unknown message type: %s", msgType)
	}

	return nil
}

// GetMeasurement returns the latest P1 meter measurement data
func (d *P1Device) GetMeasurement() (P1Measurement, error) {
	m, err := d.measurement.Get()
	if err != nil {
		return P1Measurement{}, api.ErrTimeout
	}
	return m, nil
}

// GetPowerLimits returns the battery power limits (charge, discharge in W)
func (d *P1Device) GetPowerLimits() (float64, float64, error) {
	b, err := d.batteriesData.Get()
	if err != nil {
		return 0, 0, api.ErrTimeout
	}
	return b.MaxConsumptionW, b.MaxProductionW, nil
}

// SetBatteryMode sets the battery control mode via P1 meter
func (d *P1Device) SetBatteryMode(mode string) error {
	d.log.INFO.Printf("setting battery mode to: %s", mode)

	// Try WebSocket control first
	wsMsg := map[string]any{
		"type": "batteries",
		"data": map[string]string{"mode": mode},
	}

	if err := d.conn.Send(wsMsg); err != nil {
		d.log.DEBUG.Printf("WebSocket battery control failed, falling back to HTTP: %v", err)
		return d.setBatteryModeHTTP(mode)
	} else {
		// Give the device a moment to process
		time.Sleep(100 * time.Millisecond)
		d.log.DEBUG.Println("WebSocket battery control sent")
	}

	return nil
}

// setBatteryModeHTTP sets battery mode via HTTP PUT
func (d *P1Device) setBatteryModeHTTP(mode string) error {
	uri := fmt.Sprintf("https://%s/api/batteries", d.host)

	d.log.INFO.Printf("sending HTTP PUT to %s with mode: %s", uri, mode)

	reqBody := struct {
		Mode string `json:"mode"`
	}{
		Mode: mode,
	}

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(reqBody), request.JSONEncoding)
	if err != nil {
		d.log.ERROR.Printf("failed to create HTTP request: %v", err)
		return err
	}

	// Set required headers for HomeWizard API v2
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("X-Api-Version", "2")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	var res BatteriesData
	if err := d.DoJSON(req, &res); err != nil {
		d.log.ERROR.Printf("HTTP request failed: %v", err)
		return err
	}

	d.log.INFO.Printf("battery mode set successfully via HTTP: %s (response: mode=%s, power=%.1fW)", mode, res.Mode, res.PowerW)

	return nil
}
