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
			fmt.Printf("Energy: %.0fkWh\n", energy)
		}
	}

	if v, ok := v.(api.ChargeRater); ok {
		if energy, err := v.ChargedEnergy(); err != nil {
			fmt.Printf("Charged: %v\n", err)
		} else {
			fmt.Printf("Charged: %.0fkWh\n", energy)
		}
	}

	if v, ok := v.(api.ChargeTimer); ok {
		if duration, err := v.ChargingTime(); err != nil {
			fmt.Printf("Duration: %v\n", err)
		} else {
			fmt.Printf("Duration: %v\n", duration)
		}
	}
}
