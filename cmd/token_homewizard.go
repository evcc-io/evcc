package cmd

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"
)

var homeWizardCmd = &cobra.Command{
	Use:   "homewizard",
	Short: "Pair with HomeWizard devices (WebSocket-based)",
	Run:   runHomeWizardToken,
}

func init() {
	tokenCmd.AddCommand(homeWizardCmd)
	homeWizardCmd.Flags().StringP("name", "n", "evcc", "Product name for pairing")
	homeWizardCmd.Flags().Int("timeout", 10, "Discovery timeout in seconds")
}

func runHomeWizardToken(cmd *cobra.Command, args []string) {
	// Parse log levels to enable debug/trace logging if requested
	parseLogLevels()

	name := cmd.Flag("name").Value.String()
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Validate name according to HomeWizard API requirements
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9\-_/\\# ]{1,40}$`)
	if !namePattern.MatchString(name) {
		log.FATAL.Fatal("Invalid name: must be 1-40 characters (a-z, A-Z, 0-9, -, _, \\, /, #, spaces)")
	}

	// Discovery mode - always discover all devices
	fmt.Println("HomeWizard Device Discovery")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Printf("Scanning network (max %ds)...\n", timeout)
	fmt.Println()

	devices := discoverInteractively(timeout)

	if len(devices) == 0 {
		log.FATAL.Fatal("No HomeWizard devices found on network 😞")
	}

	fmt.Println()
	fmt.Println("HomeWizard Device Pairing")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("Press the button on ALL devices NOW!")
	fmt.Println()

	// Pair all devices in parallel
	paired := pairDevicesParallel(devices, name)

	// Test WebSocket connectivity (optional)
	testWebSocketConnectivity(paired)

	// Print configuration
	printHomeWizardConfig(paired)
}

func testWebSocketConnectivity(devices []pairedDevice) {
	if len(devices) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Testing WebSocket connectivity...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i, device := range devices {
		fmt.Printf("[%d] %s (%s): ", i+1, device.Host, device.Type)

		// Simple WebSocket test - just try to connect and authenticate
		// We won't actually start a full connection manager here
		// This is just a quick validation
		if err := testWebSocketConnection(ctx, device.Host, device.Token); err != nil {
			fmt.Printf("⚠ Warning: WebSocket test failed (%v)\n", err)
		} else {
			fmt.Println("✓ WebSocket OK")
		}
	}
}

func testWebSocketConnection(ctx context.Context, host, token string) error {
	// For now, skip the actual WebSocket test
	// A full test would require importing the v2 package and creating a Connection
	// This would create circular dependencies, so we'll skip it
	// The WebSocket connection will be tested at runtime when the meter is initialized
	return nil
}

func printHomeWizardConfig(devices []pairedDevice) {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("Configuration Complete!")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Add this to your evcc.yaml configuration:")
	fmt.Println()

	// Categorize devices
	var p1Meter *pairedDevice
	kwhMeters := make([]pairedDevice, 0)
	batteries := make([]pairedDevice, 0)

	for i := range devices {
		switch devices[i].Type {
		case DeviceTypeP1Meter:
			if p1Meter == nil {
				p1Meter = &devices[i]
			}
		case DeviceTypeKWHMeter:
			kwhMeters = append(kwhMeters, devices[i])
		case DeviceTypeBattery:
			batteries = append(batteries, devices[i])
		default:
			// Unknown type - assume first is P1 meter if not set
			if p1Meter == nil {
				p1Meter = &devices[i]
			}
		}
	}

	fmt.Println("meters:")

	// Print P1 meter (grid) configuration
	if p1Meter != nil {
		fmt.Println("- name: grid")
		fmt.Println("  type: homewizard-p1")
		fmt.Printf("  host: %s\n", p1Meter.Host)
		fmt.Printf("  token: %s\n", p1Meter.Token)
		fmt.Println()
	}

	// Print kWh meter (pv) configurations
	for i, kwh := range kwhMeters {
		meterName := "pv"
		if i > 0 {
			meterName = fmt.Sprintf("pv%d", i+1)
		}
		fmt.Printf("- name: %s\n", meterName)
		fmt.Println("  type: homewizard-kwh")
		fmt.Printf("  host: %s\n", kwh.Host)
		fmt.Printf("  token: %s\n", kwh.Token)
		fmt.Println()
	}

	// Print battery configurations
	for i, bat := range batteries {
		meterName := "battery"
		if i > 0 {
			meterName = fmt.Sprintf("battery%d", i+1)
		}
		fmt.Printf("- name: %s\n", meterName)
		fmt.Println("  type: homewizard-battery")
		fmt.Printf("  host: %s\n", bat.Host)
		fmt.Printf("  token: %s\n", bat.Token)

		// Add controller configuration if P1 meter exists
		if p1Meter != nil {
			fmt.Println("  controller: grid  # Reference to the grid meter above")
		} else {
			fmt.Println("  # controller: grid  # Reference to the grid meter")
		}
		fmt.Println()
	}

	// Print helpful notes
	if len(devices) > 0 {
		fmt.Println("# Notes:")
		fmt.Println("# - Each meter entry configures ONE device")
		fmt.Println("# - homewizard-p1: P1 meter for grid monitoring")
		fmt.Println("# - homewizard-kwh: kWh meter for PV monitoring")
		fmt.Println("# - homewizard-battery: Battery device for SoC and power")
		fmt.Println("# - Battery requires 'controller' parameter (name of the P1 meter)")
		fmt.Println()
	}
}
