package wrapper

import (
	"github.com/andig/evcc/api"
)

// CompositeCharger combines Charger and ChargeController
type CompositeCharger struct {
	api.Charger
	api.ChargeController
}

// CompositeMeter combines Meter and MeterEnergy
type CompositeMeter struct {
	api.Meter
	api.MeterEnergy
}
