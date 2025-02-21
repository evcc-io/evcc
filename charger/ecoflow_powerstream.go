package charger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util/request"
)

// EcoflowPowerstream represents the Ecoflow Powerstream inverter
type EcoflowPowerstream struct {
	uri    string
	sn     string
	client *request.Helper
}

// NewEcoflowPowerstream creates a new Ecoflow Powerstream charger
func NewEcoflowPowerstream(uri, sn string) *EcoflowPowerstream {
	return &EcoflowPowerstream{
		uri:    uri,
		sn:     sn,
		client: request.NewHelper(nil),
	}
}

// SetSupplyPriority sets the power supply priority
func (c *EcoflowPowerstream) SetSupplyPriority(priority int) error {
	data := map[string]interface{}{
		"sn":      c.sn,
		"cmdCode": "WN511_SET_SUPPLY_PRIORITY_PACK",
		"params": map[string]interface{}{
			"supplyPriority": priority,
		},
	}
	return c.sendCommand(data)
}

// GetSupplyPriority gets the power supply priority
func (c *EcoflowPowerstream) GetSupplyPriority() (int, error) {
	data := map[string]interface{}{
		"sn": c.sn,
		"params": map[string]interface{}{
			"quotas": []string{"20_1.supplyPriority"},
		},
	}
	var res struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SupplyPriority int `json:"20_1.supplyPriority"`
		} `json:"data"`
	}
	err := c.getCommand(data, &res)
	return res.Data.SupplyPriority, err
}

// GetSolarPower gets the current solar power
func (c *EcoflowPowerstream) GetSolarPower() (float64, error) {
	data := map[string]interface{}{
		"sn": c.sn,
		"params": map[string]interface{}{
			"quotas": []string{"20_1.solarPower"},
		},
	}
	var res struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SolarPower float64 `json:"20_1.solarPower"`
		} `json:"data"`
	}
	err := c.getCommand(data, &res)
	return res.Data.SolarPower, err
}

// GetBatteryStatus gets the current battery status
func (c *EcoflowPowerstream) GetBatteryStatus() (float64, error) {
	data := map[string]interface{}{
		"sn": c.sn,
		"params": map[string]interface{}{
			"quotas": []string{"20_1.batteryStatus"},
		},
	}
	var res struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			BatteryStatus float64 `json:"20_1.batteryStatus"`
		} `json:"data"`
	}
	err := c.getCommand(data, &res)
	return res.Data.BatteryStatus, err
}

// SetBatteryCharge sets the battery charge level
func (c *EcoflowPowerstream) SetBatteryCharge(chargeLevel int) error {
	data := map[string]interface{}{
		"sn":      c.sn,
		"cmdCode": "WN511_SET_BATTERY_CHARGE",
		"params": map[string]interface{}{
			"chargeLevel": chargeLevel,
		},
	}
	return c.sendCommand(data)
}

// LockBattery locks the battery
func (c *EcoflowPowerstream) LockBattery() error {
	data := map[string]interface{}{
		"sn":      c.sn,
		"cmdCode": "WN511_LOCK_BATTERY",
	}
	return c.sendCommand(data)
}

// EnableGridCharging enables grid charging
func (c *EcoflowPowerstream) EnableGridCharging(enable bool) error {
	data := map[string]interface{}{
		"sn":      c.sn,
		"cmdCode": "WN511_SET_GRID_CHARGING",
		"params": map[string]interface{}{
			"enable": enable,
		},
	}
	return c.sendCommand(data)
}

// sendCommand sends a command to the Ecoflow Powerstream inverter
func (c *EcoflowPowerstream) sendCommand(data map[string]interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = c.client.Do(req)
	return err
}

// getCommand sends a GET request to the Ecoflow Powerstream inverter
func (c *EcoflowPowerstream) getCommand(data map[string]interface{}, res interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(res)
}
