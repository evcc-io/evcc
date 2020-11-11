package core

import "github.com/andig/evcc/api"

// LoadpointOptionValue defines configurable values
type LoadpointOptionValue string

// valid LoadpointOptionValues
const (
	OptionCharger LoadpointOptionValue = "charger"
	OptionMeter   LoadpointOptionValue = "meter"
)

// LoadpointOption is an optional configuration
type LoadpointOption struct {
	typ LoadpointOptionValue
	val interface{}
}

// NewLoadpointOption creates an optional site configuration
func NewLoadpointOption(key LoadpointOptionValue, val interface{}) LoadpointOption {
	return LoadpointOption{
		typ: key,
		val: val,
	}
}

func (o LoadpointOption) apply(lp *LoadPoint) {
	switch o.typ {
	case OptionCharger:
		lp.charger = o.val.(api.Charger)
	case OptionMeter:
		lp.chargeMeter = o.val.(api.Meter)
	default:
		panic("invalid option " + o.typ)
	}
}
