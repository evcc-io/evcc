package core

import "github.com/andig/evcc/api"

// SiteOptionValue defines configurable values
type SiteOptionValue string

// valid SiteOptionValues
const (
	OptionGrid    SiteOptionValue = "grid"
	OptionPV                      = "pv"
	OptionBattery                 = "battery"
)

// SiteOption is an optional configuration
type SiteOption struct {
	typ   SiteOptionValue
	meter api.Meter
}

// NewSiteOption creates an optional site configuration
func NewSiteOption(key SiteOptionValue, meter api.Meter) SiteOption {
	return SiteOption{
		typ:   key,
		meter: meter,
	}
}

func (o SiteOption) apply(site *Site) {
	switch o.typ {
	case OptionGrid:
		site.gridMeter = o.meter
	case OptionPV:
		site.pvMeter = o.meter
	case OptionBattery:
		site.batteryMeter = o.meter
	default:
		panic("invalid option " + o.typ)
	}
}
