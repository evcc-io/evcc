package cmd

import (
	"fmt"

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

	if v, ok := v.(api.ChargeRater); ok {
		if energy, err := v.ChargedEnergy(); err != nil {
			fmt.Printf("Charged: %v\n", err)
		} else {
			fmt.Printf("Charged: %.1fkWh\n", energy)
		}
	}

	if v, ok := v.(api.ChargeTimer); ok {
		if duration, err := v.ChargingTime(); err != nil {
			fmt.Printf("Duration: %v\n", err)
		} else {
			fmt.Printf("Duration: %v\n", duration)
		}
	}

	if v, ok := v.(api.ChargeRemainder); ok {
		if duration, err := v.RemainingTime(); err != nil {
			fmt.Printf("Remaining: %v\n", err)
		} else {
			fmt.Printf("Remaining: %v\n", duration)
		}
	}

	if v, ok := v.(api.Diagnosis); ok {
		fmt.Println("Diagnostic dump:")
		v.Diagnosis()
	}
}
