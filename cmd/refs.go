package cmd

import (
	"iter"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
)

var references struct {
	meter, charger, vehicle, circuit []string
}

func collectRefs(conf globalconfig.All) error {
	// site
	if err := collectSiteRefs(conf); err != nil {
		return err
	}

	// loadpoints
	if err := collectLoadpointRefs(slices.Values(conf.Loadpoints)); err != nil {
		return err
	}

	// append devices from database
	configurable, err := config.ConfigurationsByClass(templates.Loadpoint)
	if err != nil {
		return err
	}

	return collectLoadpointRefs(func(yield func(config.Named) bool) {
		for _, cc := range configurable {
			if !yield(cc.Named()) {
				return
			}
		}
	})
}

func collectSiteRefs(conf globalconfig.All) error {
	var refs struct {
		Meters core.MetersConfig `mapstructure:"meters"` // Meter references
		Other  map[string]any    `mapstructure:",remain"`
	}

	if err := util.DecodeOther(conf.Site, &refs); err != nil {
		return err
	}

	references.meter = append(references.meter, refs.Meters.GridMeterRef)
	references.meter = append(references.meter, refs.Meters.PVMetersRef...)
	references.meter = append(references.meter, refs.Meters.BatteryMetersRef...)
	references.meter = append(references.meter, refs.Meters.ExtMetersRef...)
	references.meter = append(references.meter, refs.Meters.AuxMetersRef...)

	// append devices from settings
	if v, err := settings.String(keys.GridMeter); err == nil && v != "" {
		references.meter = append(references.meter, v)
	}

	for _, key := range []string{keys.PvMeters, keys.BatteryMeters, keys.ExtMeters, keys.AuxMeters} {
		if v, err := settings.String(key); err == nil && v != "" {
			references.meter = append(references.meter, strings.Split(v, ",")...)
		}
	}

	return nil
}

func collectLoadpointRefs(named iter.Seq[config.Named]) error {
	for cc := range named {
		var refs struct {
			CircuitRef string         `mapstructure:"circuit"` // Circuit reference
			ChargerRef string         `mapstructure:"charger"` // Charger reference
			VehicleRef string         `mapstructure:"vehicle"` // Vehicle reference
			MeterRef   string         `mapstructure:"meter"`   // Charge meter reference
			Other      map[string]any `mapstructure:",remain"`
		}

		if err := util.DecodeOther(cc.Other, &refs); err != nil {
			return err
		}

		references.meter = append(references.meter, refs.MeterRef)
		references.charger = append(references.charger, refs.ChargerRef)
		references.vehicle = append(references.vehicle, refs.VehicleRef)
		references.circuit = append(references.circuit, refs.CircuitRef)
	}

	return nil
}
