//go:build integration

package charger

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
)

func TestIntegration_EcoFlowStreamCharger(t *testing.T) {
	sn := os.Getenv("ECOFLOW_SN")
	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	if sn == "" || accessKey == "" || secretKey == "" {
		t.Skip("Skipping: ECOFLOW_SN, ECOFLOW_ACCESS_KEY, ECOFLOW_SECRET_KEY required")
	}

	config := map[string]any{
		"sn":        sn,
		"accessKey": accessKey,
		"secretKey": secretKey,
		"relay":     1,
	}

	charger, err := NewEcoFlowStreamChargerFromConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create charger: %v", err)
	}

	t.Run("Status", func(t *testing.T) {
		status, err := charger.Status()
		if err != nil {
			t.Fatalf("Status() error: %v", err)
		}
		t.Logf("✅ Status: %v", status)
	})

	t.Run("Enabled", func(t *testing.T) {
		enabled, err := charger.Enabled()
		if err != nil {
			t.Fatalf("Enabled() error: %v", err)
		}
		t.Logf("✅ Enabled: %v", enabled)
	})

	t.Run("CurrentPower", func(t *testing.T) {
		if meter, ok := charger.(api.Meter); ok {
			power, err := meter.CurrentPower()
			if err != nil {
				t.Fatalf("CurrentPower() error: %v", err)
			}
			t.Logf("✅ Power: %.2f W", power)
		}
	})

	t.Run("Soc", func(t *testing.T) {
		if battery, ok := charger.(api.Battery); ok {
			soc, err := battery.Soc()
			if err != nil {
				t.Fatalf("Soc() error: %v", err)
			}
			t.Logf("✅ SOC: %.1f%%", soc)
		}
	})

	// Wait for MQTT messages
	t.Run("MQTT_Subscribe", func(t *testing.T) {
		t.Log("Waiting 10s for MQTT messages...")
		time.Sleep(10 * time.Second)

		// Check if we received MQTT data
		if c, ok := charger.(*EcoFlowStreamCharger); ok {
			c.mu.RLock()
			lastMqtt := c.lastMqtt
			data := c.data
			c.mu.RUnlock()

			if !lastMqtt.IsZero() {
				t.Logf("✅ MQTT data received at %v", lastMqtt)
				t.Logf("   SOC: %.1f%%, Power: %.1fW", data.CmsBattSoc, data.PowGetBpCms)
			} else {
				t.Log("⚠️ No MQTT data received (using REST API fallback)")
			}
		}
	})

	// Control test (only if allowed)
	if os.Getenv("ECOFLOW_ALLOW_CONTROL") == "true" {
		t.Run("Enable_Disable", func(t *testing.T) {
			// Get current state
			enabled, _ := charger.Enabled()
			t.Logf("Current state: %v", enabled)

			// Toggle
			newState := !enabled
			t.Logf("Setting to: %v", newState)

			if err := charger.Enable(newState); err != nil {
				t.Fatalf("Enable() error: %v", err)
			}

			// Wait for state change
			time.Sleep(3 * time.Second)

			// Verify
			actual, _ := charger.Enabled()
			if actual != newState {
				t.Errorf("State not changed: expected %v, got %v", newState, actual)
			} else {
				t.Logf("✅ State changed to: %v", actual)
			}

			// Restore
			t.Logf("Restoring to: %v", enabled)
			charger.Enable(enabled)
		})
	}
}
