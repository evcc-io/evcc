package core

import (
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
)

// hemsStatus is the runtime payload merged into keys.Hems alongside the
// boot-time ConfigStatus published from cmd/root.go.
type hemsStatus struct {
	Dimmed              *bool    `json:"dimmed,omitempty"`
	Curtailed           *bool    `json:"curtailed,omitempty"`
	MaxConsumptionPower float64  `json:"maxConsumptionPower,omitempty"`
	MaxProductionPower  *float64 `json:"maxProductionPower,omitempty"`
}

// publishHEMS publishes the current HEMS runtime state. The frontend
// store shallow-merges this into the boot-time ConfigStatus payload.
func (site *Site) publishHEMS() {
	if site.hems == nil {
		return
	}

	site.publish(keys.Hems, globalconfig.ConfigStatus{
		Status: hemsStatus{
			Dimmed:              site.hems.Dimmed(),
			Curtailed:           site.hems.Curtailed(),
			MaxConsumptionPower: site.hems.MaxConsumptionPower(),
			MaxProductionPower:  site.hems.MaxProductionPower(),
		},
	})
}
