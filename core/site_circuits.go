package core

import (
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
)

type circuitStruct struct {
	Title      string   `json:"title,omitempty"`
	Icon       string   `json:"icon,omitempty"`
	Power      float64  `json:"power"`
	Current    *float64 `json:"current,omitempty"`
	MaxPower   float64  `json:"maxPower,omitempty"`
	MaxCurrent float64  `json:"maxCurrent,omitempty"`
}

// publishCircuits returns a list of circuit titles
func (site *Site) publishCircuits() {
	cc := config.Circuits().Devices()
	res := make(map[string]circuitStruct, len(cc))

	for _, c := range cc {
		instance := c.Instance()
		props := deviceProperties(c)

		data := circuitStruct{
			Title:      props.Title,
			Icon:       props.Icon,
			Power:      instance.GetChargePower(),
			MaxPower:   instance.GetMaxPower(),
			MaxCurrent: instance.GetMaxCurrent(),
		}

		if instance.GetMaxCurrent() > 0 {
			data.Current = lo.EmptyableToPtr(instance.GetMaxPhaseCurrent())
		}

		res[c.Config().Name] = data
	}

	site.publish(keys.Circuits, res)
}
