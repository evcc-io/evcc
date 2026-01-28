//go:build ignore

// Test script for EcoFlow BatteryController
// Usage: go run test_battery_control.go [status|normal|hold|charge]
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/ecoflow"
)

func main() {
	sn := os.Getenv("ECOFLOW_SN")
	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	if sn == "" || accessKey == "" || secretKey == "" {
		fmt.Println("❌ Fehlende Umgebungsvariablen:")
		fmt.Println("   export ECOFLOW_SN='DEINE_SERIENNUMMER'")
		fmt.Println("   export ECOFLOW_ACCESS_KEY='DEIN_KEY'")
		fmt.Println("   export ECOFLOW_SECRET_KEY='DEIN_SECRET'")
		os.Exit(1)
	}

	cmd := "status"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	config := map[string]any{
		"sn":        sn,
		"accessKey": accessKey,
		"secretKey": secretKey,
		"usage":     "battery",
		"cache":     "5s",
	}

	fmt.Println("🔌 Verbinde mit EcoFlow...")
	meter, err := ecoflow.NewStreamFromConfig(context.Background(), config)
	if err != nil {
		fmt.Printf("❌ Fehler: %v\n", err)
		os.Exit(1)
	}

	// Warte kurz auf MQTT-Verbindung
	time.Sleep(2 * time.Second)

	// Status anzeigen
	if battery, ok := meter.(api.Battery); ok {
		soc, _ := battery.Soc()
		fmt.Printf("🔋 SOC: %.1f%%\n", soc)
	}

	if m, ok := meter.(api.Meter); ok {
		power, _ := m.CurrentPower()
		if power > 0 {
			fmt.Printf("⚡ Ladung: %.0f W\n", power)
		} else if power < 0 {
			fmt.Printf("⚡ Entladung: %.0f W\n", -power)
		} else {
			fmt.Println("⚡ Idle")
		}
	}

	bc, ok := meter.(api.BatteryController)
	if !ok {
		fmt.Println("❌ BatteryController nicht verfügbar")
		os.Exit(1)
	}

	fmt.Println("✅ BatteryController verfügbar")
	fmt.Println()

	switch cmd {
	case "status":
		fmt.Println("Verfügbare Befehle:")
		fmt.Println("  status  - Zeigt aktuellen Status")
		fmt.Println("  normal  - Normalbetrieb (Relays AN)")
		fmt.Println("  hold    - Entlade-Sperre (Relays AUS)")
		fmt.Println("  charge  - Laden aktivieren (Relays AN)")

	case "normal":
		fmt.Println("🔄 Setze BatteryNormal...")
		if err := bc.SetBatteryMode(api.BatteryNormal); err != nil {
			fmt.Printf("❌ Fehler: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ BatteryNormal aktiv - Relays eingeschaltet")

	case "hold":
		fmt.Println("🔄 Setze BatteryHold...")
		if err := bc.SetBatteryMode(api.BatteryHold); err != nil {
			fmt.Printf("❌ Fehler: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ BatteryHold aktiv - Entladung gesperrt")

	case "charge":
		fmt.Println("🔄 Setze BatteryCharge...")
		if err := bc.SetBatteryMode(api.BatteryCharge); err != nil {
			fmt.Printf("❌ Fehler: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ BatteryCharge aktiv - Laden freigegeben")
		fmt.Println("⚠️  Hinweis: Grid-Laden nicht direkt unterstützt")

	default:
		fmt.Printf("❌ Unbekannter Befehl: %s\n", cmd)
		fmt.Println("Verfügbar: status, normal, hold, charge")
		os.Exit(1)
	}
}
