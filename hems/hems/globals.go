package hems

import "github.com/evcc-io/evcc/api"

// Dimmed reports nil until MaxConsumptionPower is known (see api.HEMS).
func Dimmed(hems api.HEMS) *bool {
	if hems == nil {
		return nil
	}

	dimmed := hems.MaxConsumptionPower()
	if dimmed == nil {
		return nil
	}

	return new(*dimmed > 0)
}

func Curtailed(hems api.HEMS) *bool {
	if hems == nil {
		return nil
	}

	if percent := hems.CurtailedPercent(); percent != nil {
		return new(*percent < 100)
	}

	return nil
}
