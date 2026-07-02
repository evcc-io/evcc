package hems

import "github.com/evcc-io/evcc/api"

func Dimmed(hems api.HEMS) *bool {
	dimmed := hems.MaxConsumptionPower()
	if dimmed == nil {
		return nil
	}

	return new(*dimmed > 0)
}
