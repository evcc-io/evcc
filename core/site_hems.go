package core

import (
	"github.com/evcc-io/evcc/core/keys"
)

// hemsStatus is the per-tick runtime payload merged into state.hems.status
// alongside the boot-time ConfigStatus published from cmd/root.go. Published
// under a dotted key so the per-tick frame does not overwrite the cached
// boot-time {config, yamlSource} entry on new WS subscribers.
type hemsStatus struct {
	Dimmed              *bool    `json:"dimmed,omitempty"`
	Curtailed           *bool    `json:"curtailed,omitempty"`
	MaxConsumptionPower float64  `json:"maxConsumptionPower,omitempty"`
	MaxProductionPower  *float64 `json:"maxProductionPower,omitempty"`
}

// publishHEMS publishes the current HEMS runtime state to state.hems.status.
func (site *Site) publishHEMS() {
	if site.hems == nil {
		return
	}

	site.publish(keys.Hems+".status", hemsStatus{
		Dimmed:              site.hems.Dimmed(),
		Curtailed:           site.hems.Curtailed(),
		MaxConsumptionPower: site.hems.MaxConsumptionPower(),
		MaxProductionPower:  site.hems.MaxProductionPower(),
	})
}
