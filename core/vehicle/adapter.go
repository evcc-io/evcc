package vehicle

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
)

var _ API = (*adapter)(nil)

// Publish publishes vehicle updates at site level
var Publish func()

type adapter struct {
	log         *util.Logger
	name        string
	api.Vehicle // TODO handle instance updates
}

func (v *adapter) key() string {
	return fmt.Sprintf("vehicle.%s.", v.name)
}

func (v *adapter) publish() {
	if Publish != nil {
		Publish()
	}
}

func (v *adapter) Instance() api.Vehicle {
	return v.Vehicle
}

func (v *adapter) Name() string {
	return v.name
}

// GetMode returns the mode
func (v *adapter) GetMode() api.ChargeMode {
	if v, err := settings.String(v.key() + keys.Mode); err == nil {
		return api.ChargeMode(v)
	}
	return api.ChargeMode("")
}

// SetMode sets the mode
func (v *adapter) SetMode(mode api.ChargeMode) {
	v.log.DEBUG.Printf("set %s mode: %s", v.name, mode)
	settings.SetString(v.key()+keys.Mode, string(mode))
	v.publish()
}

// GetPhases returns the phases
func (v *adapter) GetPhases() int {
	if v, err := settings.Int(v.key() + keys.PhasesConfigured); err == nil {
		return int(v)
	}
	return 0
}

// SetPhases sets the phases
func (v *adapter) SetPhases(phases int) {
	v.log.DEBUG.Printf("set %s phases: %d", v.name, phases)
	settings.SetInt(v.key()+keys.PhasesConfigured, int64(phases))
	v.publish()
}

// GetPriority returns the priority
func (v *adapter) GetPriority() int {
	if v, err := settings.Int(v.key() + keys.Priority); err == nil {
		return int(v)
	}
	return 0
}

// SetPriority sets the priority
func (v *adapter) SetPriority(prio int) {
	v.log.DEBUG.Printf("set %s priority: %d", v.name, prio)
	settings.SetInt(v.key()+keys.Priority, int64(prio))
	v.publish()
}

// GetMinSoc returns the min soc
func (v *adapter) GetMinSoc() int {
	if v, err := settings.Int(v.key() + keys.MinSoc); err == nil {
		return int(v)
	}
	return 0
}

// SetMinSoc sets the min soc
func (v *adapter) SetMinSoc(soc int) {
	v.log.DEBUG.Printf("set %s min soc: %d", v.name, soc)
	settings.SetInt(v.key()+keys.MinSoc, int64(soc))
	v.publish()
}

// GetLimitSoc returns the limit soc
func (v *adapter) GetLimitSoc() int {
	if v, err := settings.Int(v.key() + keys.LimitSoc); err == nil {
		return int(v)
	}
	return 0
}

// SetLimitSoc sets the limit soc
func (v *adapter) SetLimitSoc(soc int) {
	v.log.DEBUG.Printf("set %s limit soc: %d", v.name, soc)
	settings.SetInt(v.key()+keys.LimitSoc, int64(soc))
	v.publish()
}

// GetPlanSoc returns the charge plan soc
func (v *adapter) GetPlanSoc() (time.Time, int) {
	var ts time.Time
	if v, err := settings.Time(v.key() + keys.PlanTime); err == nil {
		ts = v
	}
	var soc int
	if v, err := settings.Int(v.key() + keys.PlanSoc); err == nil {
		soc = int(v)
	}
	return ts, soc
}

// SetPlanSoc sets the charge plan soc
func (v *adapter) SetPlanSoc(ts time.Time, soc int) error {
	if !ts.IsZero() && ts.Before(time.Now()) {
		return errors.New("timestamp is in the past")
	}

	// remove plan
	if soc == 0 {
		ts = time.Time{}
	}

	v.log.DEBUG.Printf("set %s plan soc: %d @ %v", v.name, soc, ts.Round(time.Second).Local())

	settings.SetTime(v.key()+keys.PlanTime, ts)
	settings.SetInt(v.key()+keys.PlanSoc, int64(soc))

	v.publish()

	return nil
}

// GetMinCurrent returns the min current
func (v *adapter) GetMinCurrent() float64 {
	if v, err := settings.Float(v.key() + keys.MinCurrent); err == nil {
		return v
	}
	return 0
}

// SetMinCurrent sets the min current
func (v *adapter) SetMinCurrent(current float64) {
	v.log.DEBUG.Printf("set %s min current: %g", v.name, current)
	settings.SetFloat(v.key()+keys.MinCurrent, current)
	v.publish()
}

// GetMaxCurrent returns the max current
func (v *adapter) GetMaxCurrent() float64 {
	if v, err := settings.Float(v.key() + keys.MaxCurrent); err == nil {
		return v
	}
	return 0
}

// SetMaxCurrent sets the max current
func (v *adapter) SetMaxCurrent(current float64) {
	v.log.DEBUG.Printf("set %s max current: %g", v.name, current)
	settings.SetFloat(v.key()+keys.MaxCurrent, current)
	v.publish()
}
