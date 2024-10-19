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

var _ API = (*AdapterStruct)(nil)

// Publish publishes vehicle updates at site level
var Publish func()

type AdapterStruct struct {
	log         *util.Logger
	name        string
	api.Vehicle // TODO handle instance updates
}

func (v *AdapterStruct) key() string {
	return fmt.Sprintf("vehicle.%s.", v.name)
}

func (v *AdapterStruct) publish() {
	if Publish != nil {
		Publish()
	}
}

func (v *AdapterStruct) Instance() api.Vehicle {
	return v.Vehicle
}

func (v *AdapterStruct) Name() string {
	return v.name
}

// GetMinSoc returns the min soc
func (v *AdapterStruct) GetMinSoc() int {
	if v, err := settings.Int(v.key() + keys.MinSoc); err == nil {
		return int(v)
	}
	return 0
}

// SetMinSoc sets the min soc
func (v *AdapterStruct) SetMinSoc(soc int) {
	v.log.DEBUG.Printf("set %s min soc: %d", v.name, soc)
	settings.SetInt(v.key()+keys.MinSoc, int64(soc))
	v.publish()
}

// GetLimitSoc returns the limit soc
func (v *AdapterStruct) GetLimitSoc() int {
	if v, err := settings.Int(v.key() + keys.LimitSoc); err == nil {
		return int(v)
	}
	return 0
}

// SetLimitSoc sets the limit soc
func (v *AdapterStruct) SetLimitSoc(soc int) {
	v.log.DEBUG.Printf("set %s limit soc: %d", v.name, soc)
	settings.SetInt(v.key()+keys.LimitSoc, int64(soc))
	v.publish()
}

// GetPlanSoc returns the charge plan soc
func (v *AdapterStruct) GetPlanSoc() (time.Time, int) {
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
func (v *AdapterStruct) SetPlanSoc(ts time.Time, soc int) error {
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

func (v *AdapterStruct) SetRepeatingPlans(plans []api.RepeatingPlanStruct) error {
	v.log.DEBUG.Printf("update repeating plans for %s to: %v", v.name, plans)

	settings.SetJson(v.key()+keys.RepeatingPlans, plans)

	v.publish()

	return nil
}

func (v *AdapterStruct) GetRepeatingPlans() []api.RepeatingPlanStruct {
	var plans []api.RepeatingPlanStruct

	err := settings.Json(v.key()+keys.RepeatingPlans, &plans)
	if err == nil {
		return plans
	}

	v.log.DEBUG.Printf("update repeating plans triggered error: %s", err)

	return []api.RepeatingPlanStruct{}
}

func (v *AdapterStruct) GetRepeatingPlansWithTimestamps() []api.PlanStruct {
	var formattedPlans []api.PlanStruct

	plans := v.GetRepeatingPlans()

	for _, p := range plans {
		formattedPlans = append(formattedPlans, p.ToPlansWithTimestamp()...)
	}

	return formattedPlans
}
