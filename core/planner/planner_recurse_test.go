package planner

import (
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

// TestCase35kWh_4h45m tests the specific scenario from the UI
// 35 kWh, 4:45h duration, 7.2 kW, maxChargingWindows=2
func TestCase35kWh_4h45m(t *testing.T) {
	log := util.NewLogger("test")

	// Parse die komplette Tarifstruktur vom 14.10.2025
	rates := parseTestRates()

	// Mock-Tariff mit gomock
	ctrl := gomock.NewController(t)
	mockTariff := api.NewMockTariff(ctrl)
	mockTariff.EXPECT().Rates().AnyTimes().Return(rates, nil)

	// Erstelle Planner mit Mock-Clock
	mockClock := clock.NewMock()
	// Setze aktuelle Zeit auf 14.10.2025 23:50 (kurz vor Mitternacht)
	mockClock.Set(time.Date(2025, 10, 14, 23, 50, 0, 0, time.UTC))

	planner := New(log, mockTariff, func(p *Planner) {
		p.clock = mockClock
	})

	// Parameter
	powerKW := 7.2
	energyKWh := 35.0

	// Berechne tatsächliche Ladezeit (nicht Kalenderzeit!)
	// Energy = Power * Time => 35 kWh = 7.2 kW * t => t = 4.861 hours
	actualChargingDuration := time.Duration(float64(time.Hour) * energyKWh / powerKW)
	fmt.Printf("Energie: %.1f kWh\n", energyKWh)
	fmt.Printf("Leistung: %.1f kW\n", powerKW)
	fmt.Printf("Tatsächliche Ladezeit: %v (%.2f hours)\n", actualChargingDuration, actualChargingDuration.Hours())
	fmt.Printf("UI zeigt: 4:45h\n\n")

	// Target: morgen 00:00 (in 10 Minuten)
	targetTime := time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC)
	precondition := 0 * time.Minute

	// Test mit Mode 2 (2 Ladefenster)
	fmt.Println("=== MODE 2: maxChargingWindows=2 ===")
	plan2 := planner.Plan(actualChargingDuration, precondition, targetTime, 2)
	analyzeAndPrintPlan(plan2, rates, "MODE 2")

	// Test mit Mode 1 (kontinuierliches Fenster)
	fmt.Println("\n=== MODE 1: maxChargingWindows=1 ===")
	plan1 := planner.Plan(actualChargingDuration, precondition, targetTime, 1)
	analyzeAndPrintPlan(plan1, rates, "MODE 1")

	// Test mit Standard (alle Fenster)
	fmt.Println("\n=== STANDARD: maxChargingWindows=0 ===")
	plan0 := planner.Plan(actualChargingDuration, precondition, targetTime)
	analyzeAndPrintPlan(plan0, rates, "STANDARD")

	// Vergleich
	fmt.Println("\n=== VERGLEICH ===")
	cost0 := calculatePlanCost(plan0, actualChargingDuration)
	cost1 := calculatePlanCost(plan1, actualChargingDuration)
	cost2 := calculatePlanCost(plan2, actualChargingDuration)

	fmt.Printf("Standard (all windows): €%.2f\n", cost0)
	fmt.Printf("Mode 1 (1 window):      €%.2f\n", cost1)
	fmt.Printf("Mode 2 (2 windows):     €%.2f\n", cost2)

	if cost2 > cost1 {
		t.Logf("⚠️  MODE 2 ist TEURER als MODE 1! Mode 2: €%.2f > Mode 1: €%.2f", cost2, cost1)
	}

	// Detaillierte Analyse
	fmt.Println("\n=== DETAILLIERTE FENSTER-ANALYSE ===")
	fmt.Printf("Mode 1: %d Fenster, Cost €%.2f\n", len(plan1), cost1)
	for i, slot := range plan1 {
		fmt.Printf("  [%d] %s - %s (%.3f €/kWh)\n", i+1, slot.Start.Format("15:04"), slot.End.Format("15:04"), slot.Value)
	}

	fmt.Printf("\nMode 2: %d Fenster, Cost €%.2f\n", len(plan2), cost2)
	for i, slot := range plan2 {
		fmt.Printf("  [%d] %s - %s (%.3f €/kWh)\n", i+1, slot.Start.Format("15:04"), slot.End.Format("15:04"), slot.Value)
	}
}

func analyzeAndPrintPlan(plan api.Rates, fullRates api.Rates, title string) {
	if len(plan) == 0 {
		fmt.Printf("%s: LEER\n", title)
		return
	}

	fmt.Printf("%s: %d Fenster\n", title, len(plan))

	var totalDuration time.Duration
	var totalCost float64
	minPrice := 999.0
	maxPrice := 0.0

	for i, slot := range plan {
		duration := slot.End.Sub(slot.Start)
		totalDuration += duration
		cost := slot.Value * duration.Hours()
		totalCost += cost

		if slot.Value < minPrice {
			minPrice = slot.Value
		}
		if slot.Value > maxPrice {
			maxPrice = slot.Value
		}

		fmt.Printf("  [%d] %s - %s (%.3f €/kWh, %v, €%.2f)\n",
			i+1,
			slot.Start.Format("15:04"),
			slot.End.Format("15:04"),
			slot.Value,
			duration,
			cost,
		)
	}

	fmt.Printf("  Gesamt: %v (%.2f h), Kosten: €%.2f, Ø-Preis: €%.3f/kWh\n",
		totalDuration,
		totalDuration.Hours(),
		totalCost,
		totalCost/totalDuration.Hours())
	fmt.Printf("  Preisspanne: €%.3f - €%.3f/kWh\n", minPrice, maxPrice)
}

func calculatePlanCost(plan api.Rates, requiredDuration time.Duration) float64 {
	if len(plan) == 0 {
		return 0
	}

	var cost float64
	for _, slot := range plan {
		duration := slot.End.Sub(slot.Start)
		cost += slot.Value * duration.Hours()
	}
	return cost
}

func parseTestRates() api.Rates {
	type rateData struct {
		from  string
		to    string
		price float64
	}

	data := []rateData{
		{"2025-10-14 00:00:00", "2025-10-14 00:15:00", 0.288},
		{"2025-10-14 00:15:00", "2025-10-14 00:30:00", 0.290},
		{"2025-10-14 00:30:00", "2025-10-14 00:45:00", 0.277},
		{"2025-10-14 00:45:00", "2025-10-14 01:00:00", 0.270},
		{"2025-10-14 01:00:00", "2025-10-14 01:15:00", 0.279},
		{"2025-10-14 01:15:00", "2025-10-14 01:30:00", 0.276},
		{"2025-10-14 01:30:00", "2025-10-14 01:45:00", 0.274},
		{"2025-10-14 01:45:00", "2025-10-14 02:00:00", 0.267},
		{"2025-10-14 02:00:00", "2025-10-14 02:15:00", 0.276},
		{"2025-10-14 02:15:00", "2025-10-14 02:30:00", 0.271},
		{"2025-10-14 02:30:00", "2025-10-14 02:45:00", 0.268},
		{"2025-10-14 02:45:00", "2025-10-14 03:00:00", 0.264},
		{"2025-10-14 03:00:00", "2025-10-14 03:15:00", 0.271},
		{"2025-10-14 03:15:00", "2025-10-14 03:30:00", 0.269},
		{"2025-10-14 03:30:00", "2025-10-14 03:45:00", 0.267},
		{"2025-10-14 03:45:00", "2025-10-14 04:00:00", 0.268},
		{"2025-10-14 04:00:00", "2025-10-14 04:15:00", 0.268},
		{"2025-10-14 04:15:00", "2025-10-14 04:30:00", 0.269},
		{"2025-10-14 04:30:00", "2025-10-14 04:45:00", 0.269},
		{"2025-10-14 04:45:00", "2025-10-14 05:00:00", 0.276},
		{"2025-10-14 05:00:00", "2025-10-14 05:15:00", 0.267},
		{"2025-10-14 05:15:00", "2025-10-14 05:30:00", 0.274},
		{"2025-10-14 05:30:00", "2025-10-14 05:45:00", 0.273},
		{"2025-10-14 05:45:00", "2025-10-14 06:00:00", 0.275},
		{"2025-10-14 06:00:00", "2025-10-14 06:15:00", 0.262},
		{"2025-10-14 06:15:00", "2025-10-14 06:30:00", 0.296},
		{"2025-10-14 06:30:00", "2025-10-14 06:45:00", 0.322},
		{"2025-10-14 06:45:00", "2025-10-14 07:00:00", 0.313},
		{"2025-10-14 07:00:00", "2025-10-14 07:15:00", 0.307},
		{"2025-10-14 07:15:00", "2025-10-14 07:30:00", 0.322},
		{"2025-10-14 07:30:00", "2025-10-14 07:45:00", 0.364},
		{"2025-10-14 07:45:00", "2025-10-14 08:00:00", 0.375},
		{"2025-10-14 08:00:00", "2025-10-14 08:15:00", 0.426},
		{"2025-10-14 08:15:00", "2025-10-14 08:30:00", 0.418},
		{"2025-10-14 08:30:00", "2025-10-14 08:45:00", 0.350},
		{"2025-10-14 08:45:00", "2025-10-14 09:00:00", 0.319},
		{"2025-10-14 09:00:00", "2025-10-14 09:15:00", 0.408},
		{"2025-10-14 09:15:00", "2025-10-14 09:30:00", 0.334},
		{"2025-10-14 09:30:00", "2025-10-14 09:45:00", 0.311},
		{"2025-10-14 09:45:00", "2025-10-14 10:00:00", 0.297},
		{"2025-10-14 10:00:00", "2025-10-14 10:15:00", 0.366},
		{"2025-10-14 10:15:00", "2025-10-14 10:30:00", 0.309},
		{"2025-10-14 10:30:00", "2025-10-14 10:45:00", 0.290},
		{"2025-10-14 10:45:00", "2025-10-14 11:00:00", 0.281},
		{"2025-10-14 11:00:00", "2025-10-14 11:15:00", 0.312},
		{"2025-10-14 11:15:00", "2025-10-14 11:30:00", 0.295},
		{"2025-10-14 11:30:00", "2025-10-14 11:45:00", 0.289},
		{"2025-10-14 11:45:00", "2025-10-14 12:00:00", 0.279},
		{"2025-10-14 12:00:00", "2025-10-14 12:15:00", 0.286},
		{"2025-10-14 12:15:00", "2025-10-14 12:30:00", 0.284},
		{"2025-10-14 12:30:00", "2025-10-14 12:45:00", 0.279},
		{"2025-10-14 12:45:00", "2025-10-14 13:00:00", 0.269},
		{"2025-10-14 13:00:00", "2025-10-14 13:15:00", 0.276},
		{"2025-10-14 13:15:00", "2025-10-14 13:30:00", 0.273},
		{"2025-10-14 13:30:00", "2025-10-14 13:45:00", 0.275},
		{"2025-10-14 13:45:00", "2025-10-14 14:00:00", 0.272},
		{"2025-10-14 14:00:00", "2025-10-14 14:15:00", 0.254},
		{"2025-10-14 14:15:00", "2025-10-14 14:30:00", 0.271},
		{"2025-10-14 14:30:00", "2025-10-14 14:45:00", 0.281},
		{"2025-10-14 14:45:00", "2025-10-14 15:00:00", 0.301},
		{"2025-10-14 15:00:00", "2025-10-14 15:15:00", 0.273},
		{"2025-10-14 15:15:00", "2025-10-14 15:30:00", 0.286},
		{"2025-10-14 15:30:00", "2025-10-14 15:45:00", 0.292},
		{"2025-10-14 15:45:00", "2025-10-14 16:00:00", 0.337},
		{"2025-10-14 16:00:00", "2025-10-14 16:15:00", 0.260},
		{"2025-10-14 16:15:00", "2025-10-14 16:30:00", 0.289},
		{"2025-10-14 16:30:00", "2025-10-14 16:45:00", 0.336},
		{"2025-10-14 16:45:00", "2025-10-14 17:00:00", 0.418},
		{"2025-10-14 17:00:00", "2025-10-14 17:15:00", 0.287},
		{"2025-10-14 17:15:00", "2025-10-14 17:30:00", 0.324},
		{"2025-10-14 17:30:00", "2025-10-14 17:45:00", 0.436},
		{"2025-10-14 17:45:00", "2025-10-14 18:00:00", 0.602},
		{"2025-10-14 18:00:00", "2025-10-14 18:15:00", 0.478},
		{"2025-10-14 18:15:00", "2025-10-14 18:30:00", 0.582},
		{"2025-10-14 18:30:00", "2025-10-14 18:45:00", 0.716},
		{"2025-10-14 18:45:00", "2025-10-14 19:00:00", 0.756},
		{"2025-10-14 19:00:00", "2025-10-14 19:15:00", 0.739},
		{"2025-10-14 19:15:00", "2025-10-14 19:30:00", 0.685},
		{"2025-10-14 19:30:00", "2025-10-14 19:45:00", 0.604},
		{"2025-10-14 19:45:00", "2025-10-14 20:00:00", 0.516},
		{"2025-10-14 20:00:00", "2025-10-14 20:15:00", 0.547},
		{"2025-10-14 20:15:00", "2025-10-14 20:30:00", 0.440},
		{"2025-10-14 20:30:00", "2025-10-14 20:45:00", 0.368},
		{"2025-10-14 20:45:00", "2025-10-14 21:00:00", 0.324},
		{"2025-10-14 21:00:00", "2025-10-14 21:15:00", 0.406},
		{"2025-10-14 21:15:00", "2025-10-14 21:30:00", 0.348},
		{"2025-10-14 21:30:00", "2025-10-14 21:45:00", 0.326},
		{"2025-10-14 21:45:00", "2025-10-14 22:00:00", 0.301},
		{"2025-10-14 22:00:00", "2025-10-14 22:15:00", 0.326},
		{"2025-10-14 22:15:00", "2025-10-14 22:30:00", 0.309},
		{"2025-10-14 22:30:00", "2025-10-14 22:45:00", 0.298},
		{"2025-10-14 22:45:00", "2025-10-14 23:00:00", 0.290},
		{"2025-10-14 23:00:00", "2025-10-14 23:15:00", 0.299},
		{"2025-10-14 23:15:00", "2025-10-14 23:30:00", 0.292},
		{"2025-10-14 23:30:00", "2025-10-14 23:45:00", 0.283},
		{"2025-10-14 23:45:00", "2025-10-15 00:00:00", 0.281},
	}

	rates := make(api.Rates, len(data))
	for i, d := range data {
		start, _ := time.Parse("2006-01-02 15:04:05", d.from)
		end, _ := time.Parse("2006-01-02 15:04:05", d.to)
		rates[i] = api.Rate{Start: start, End: end, Value: d.price}
	}

	return rates
}
