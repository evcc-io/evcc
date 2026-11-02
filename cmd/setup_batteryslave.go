package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/util/config"
)

type meterConfig struct {
	name   string
	values map[string]any
}

type batteryConfigWarning struct {
	Device string `json:"device"`
	Master string `json:"master"`
	Reason string `json:"reason"`
}

const (
	batteryConfigWarningInvalidMaster    = "invalidMaster"
	batteryConfigWarningMasterIsSlave    = "masterIsSlave"
	batteryConfigWarningUnsupportedSlave = "unsupportedSlave"
)

func combineBatteryMeterConfigs(static []config.Named, configurable []config.Config) []meterConfig {
	meters := make([]meterConfig, 0, len(static)+len(configurable))
	for i := range static {
		meters = append(meters, meterConfig{name: static[i].Name, values: static[i].Other})
	}
	for i := range configurable {
		meters = append(meters, meterConfig{
			name:   config.NameForID(configurable[i].ID),
			values: configurable[i].Data,
		})
	}
	return meters
}

func clearMastersOfUncontrollableBatterySlaves(meters []meterConfig, batteryNames []string) []batteryConfigWarning {
	var warnings []batteryConfigWarning
	for _, meter := range meters {
		if !slices.Contains(batteryNames, meter.name) {
			continue
		}

		dev, err := config.Meters().ByName(meter.name)
		if err != nil || dev.Instance() == nil {
			continue
		}
		if warning := clearMasterOfUncontrollableBatterySlave(batteryMeterTitleOrName(dev), meter.values, dev.Instance()); warning != nil {
			warnings = append(warnings, *warning)
		}
	}
	return warnings
}

func clearMasterOfUncontrollableBatterySlave(name string, values map[string]any, meter api.Meter) *batteryConfigWarning {
	key, masterValue := batteryMasterProperty(values)
	master, _ := masterValue.(string)
	if key == "" || master == "" || core.SupportsBatterySlaveControl(meter) {
		return nil
	}

	log.WARN.Printf("battery %s: cannot be a slave, missing charge/discharge power control; ignoring master %s", name, master)
	values[key] = nil
	return &batteryConfigWarning{Device: name, Master: master, Reason: batteryConfigWarningUnsupportedSlave}
}

func batteryMeterTitleOrName(dev config.Device[api.Meter]) string {
	if configurable, ok := dev.(config.ConfigurableDevice[api.Meter]); ok && configurable.Properties().Title != "" {
		return configurable.Properties().Title
	}
	return dev.Config().Name
}

func batteryMasterProperty(values map[string]any) (string, any) {
	for field, value := range values {
		if strings.EqualFold(field, "master") {
			return field, value
		}
	}
	return "", nil
}

func normalizeBatteryMasters(meters []meterConfig, batteryNames []string) []batteryConfigWarning {
	var warnings []batteryConfigWarning
	isBattery := func(name string) bool {
		return slices.Contains(batteryNames, name) && slices.ContainsFunc(meters, func(meter meterConfig) bool {
			return meter.name == name
		})
	}

	for _, meter := range meters {
		if !isBattery(meter.name) {
			continue
		}

		key, master := batteryMasterProperty(meter.values)
		if key == "" || master == nil {
			continue
		}

		id, ok := master.(string)
		if ok && id == "" {
			continue
		}
		if ok && id != meter.name {
			if isBattery(id) {
				continue
			}
		}

		log.WARN.Printf("battery %s: invalid master %v, ignoring", meter.name, master)
		meter.values[key] = nil
		warnings = append(warnings, batteryConfigWarning{
			Device: meterConfigTitleOrName(meter),
			Master: fmt.Sprint(master),
			Reason: batteryConfigWarningInvalidMaster,
		})
	}

	// Snapshot parent links before clearing nested assignments, independent of input order.
	mastersWithMaster := make(map[string]bool)
	for _, meter := range meters {
		if !isBattery(meter.name) {
			continue
		}
		_, master := batteryMasterProperty(meter.values)
		if id, ok := master.(string); ok && id != "" {
			mastersWithMaster[meter.name] = true
		}
	}

	for _, meter := range meters {
		if !isBattery(meter.name) {
			continue
		}
		key, master := batteryMasterProperty(meter.values)
		id, ok := master.(string)
		if key == "" || !ok || id == "" || !mastersWithMaster[id] {
			continue
		}

		log.WARN.Printf("battery %s: invalid master %s, master is already a slave; ignoring", meter.name, id)
		meter.values[key] = nil
		warnings = append(warnings, batteryConfigWarning{
			Device: meterConfigTitleOrName(meter),
			Master: id,
			Reason: batteryConfigWarningMasterIsSlave,
		})
	}
	return warnings
}

func meterConfigTitleOrName(meter meterConfig) string {
	if title, ok := meter.values["deviceTitle"].(string); ok && title != "" {
		return title
	}
	return meter.name
}
