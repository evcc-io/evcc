package hems

import "github.com/evcc-io/evcc/api"

// DimLimit reports nil until MaxConsumptionPower is known (see api.HEMS).
func DimLimit(hems api.HEMS) *float64 {
	if hems == nil {
		return nil
	}

	return hems.MaxConsumptionPower()
}

// Dimmed reports nil until MaxConsumptionPower is known (see api.HEMS).
func Dimmed(hems api.HEMS) *bool {
	limit := DimLimit(hems)
	if limit == nil {
		return nil
	}

	return new(*limit > 0)
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
