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
