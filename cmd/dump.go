package cmd

import (
	"fmt"
	"math"
	"time"

	"github.com/andig/evcc/api"
)

var truefalse = map[bool]string{false: "false", true: "true"}

func dumpAPIs(v interface{}) {
	if v, ok := v.(api.Meter); ok {
		if power, err := v.CurrentPower(); err != nil {
			fmt.Printf("Power: %v\n", err)
		} else {
			fmt.Printf("Power: %.0fW\n", power)
		}
	}

	if v, ok := v.(api.MeterEnergy); ok {
		if energy, err := v.TotalEnergy(); err != nil {
			fmt.Printf("Energy: %v\n", err)
		} else {
			fmt.Printf("Energy: %.1fkWh\n", energy)
		}
	}

	if v, ok := v.(api.MeterCurrent); ok {
		if i1, i2, i3, err := v.Currents(); err != nil {
			fmt.Printf("Current L1..L3: %v\n", err)
		} else {
			fmt.Printf("Current L1..L3: %.1fA %.1fA %.1fA\n", i1, i2, i3)
		}
	}

	if v, ok := v.(api.Battery); ok {
		if soc, err := v.SoC(); err != nil {
			fmt.Printf("SoC: %v\n", err)
		} else {
			fmt.Printf("SoC: %.0f%%\n", soc)
		}
	}

	if v, ok := v.(api.ChargeRater); ok {
		if energy, err := v.ChargedEnergy(); err != nil {
			fmt.Printf("Charged: %v\n", err)
		} else {
			fmt.Printf("Charged: %.1fkWh\n", energy)
		}
	}

	if v, ok := v.(api.VehicleStatus); ok {
		if status, err := v.Status(); err != nil {
			fmt.Printf("Charge status: %v\n", err)
		} else {
			fmt.Printf("Charge status: %v\n", status)
		}
	}

	if v, ok := v.(api.ChargeTimer); ok {
		if duration, err := v.ChargingTime(); err != nil {
			fmt.Printf("Duration: %v\n", err)
		} else {
			fmt.Printf("Duration: %v\n", duration.Truncate(time.Second))
		}
	}

	if v, ok := v.(api.ChargeFinishTimer); ok {
		if ft, err := v.FinishTime(); err != nil {
			fmt.Printf("Finish time: %v\n", err)
		} else {
			fmt.Printf("Finish time: %v\n", ft.Truncate(time.Minute))
		}
	}

	if v, ok := v.(api.Climater); ok {
		if active, ot, tt, err := v.Climater(); err != nil {
			fmt.Printf("Climater: %v\n", err)
		} else {
			fmt.Printf("Climate active: %v\n", active)
			if !math.IsNaN(ot) {
				fmt.Printf("Outside temp: %.1f°C\n", ot)
			}
			if !math.IsNaN(tt) {
				fmt.Printf("Target temp: %.1f°C\n", tt)
			}
		}
	}

	if v, ok := v.(api.Diagnosis); ok {
		fmt.Println("Diagnostic dump:")
		v.Diagnosis()
	}
}
