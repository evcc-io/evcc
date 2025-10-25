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
func (v *adapter) GetPlanSoc() (time.Time, time.Duration, int) {
	var ts time.Time
	if v, err := settings.Time(v.key() + keys.PlanTime); err == nil {
		ts = v
	}
	var precondition time.Duration
	if v, err := settings.Int(v.key() + keys.PlanPrecondition); err == nil {
		precondition = time.Duration(v) * time.Second
	}
	var soc int
	if v, err := settings.Int(v.key() + keys.PlanSoc); err == nil {
		soc = int(v)
	}
	return ts, precondition, soc
}

// SetPlanSoc sets the charge plan soc
func (v *adapter) SetPlanSoc(ts time.Time, precondition time.Duration, soc int) error {
	if !ts.IsZero() && ts.Before(time.Now()) {
		return errors.New("timestamp is in the past")
	}

	// remove plan
	if soc == 0 {
		ts = time.Time{}
		v.log.DEBUG.Printf("delete %s plan", v.name)
	} else {
		v.log.DEBUG.Printf("set %s plan soc: %d @ %v (precondition: %v)", v.name, soc, ts.Round(time.Second).Local(), precondition)
	}

	settings.SetTime(v.key()+keys.PlanTime, ts)
	settings.SetInt(v.key()+keys.PlanPrecondition, int64(precondition.Seconds()))
	settings.SetInt(v.key()+keys.PlanSoc, int64(soc))

	v.publish()

	return nil
}

func (v *adapter) SetRepeatingPlans(plans []api.RepeatingPlan) error {
	for _, plan := range plans {
		for _, day := range plan.Weekdays {
			if day < 0 || day > 6 {
				return fmt.Errorf("weekday out of range: %v", day)
			}
		}
		if _, err := time.LoadLocation(plan.Tz); err != nil {
			return fmt.Errorf("invalid timezone: %v", err)
		}
		if _, err := time.Parse("15:04", plan.Time); err != nil {
			return fmt.Errorf("invalid time: %v", err)
		}
	}

	v.log.DEBUG.Printf("update repeating plans for %s to: %v", v.name, plans)

	settings.SetJson(v.key()+keys.RepeatingPlans, plans)

	v.publish()

	return nil
}

func (v *adapter) GetRepeatingPlans() []api.RepeatingPlan {
	var plans []api.RepeatingPlan

	if err := settings.Json(v.key()+keys.RepeatingPlans, &plans); err != nil {
		return nil
	}

	return plans
}
