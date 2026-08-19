package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type RepeatingPlan struct {
	Weekdays []int        `json:"weekdays"`          // 0-6 (Sunday-Saturday)
	Time     string       `json:"time"`              // HH:MM
	Tz       string       `json:"tz"`                // timezone in IANA format
	Soc      int          `json:"soc"`               // target soc, optional if absence is set
	Active   bool         `json:"active"`            // active flag
	Absence  *PlanAbsence `json:"absence,omitempty"` // absence following the plan time
}

// PlanGoal describes a single charging goal for multi-goal planning
type PlanGoal struct {
	Duration time.Duration // required cumulative charging duration from now to reach the goal by Time
	Time     time.Time     // target time
	Absence  time.Duration // duration of absence following target time- no charging during absence
}

// PlanAbsence describes an absence following a plan's target time during which
// the vehicle cannot charge and its soc drops, e.g. a trip
type PlanAbsence struct {
	Duration time.Duration `json:"duration"` // absence duration
	Soc      int           `json:"soc"`      // expected soc drop during absence in %
}

type planAbsence struct {
	Duration int64 `json:"duration"` // absence duration in seconds
	Soc      int   `json:"soc"`      // expected soc drop during absence in %
}

func (pa PlanAbsence) MarshalJSON() ([]byte, error) {
	return json.Marshal(planAbsence{
		Duration: int64(pa.Duration.Seconds()),
		Soc:      pa.Soc,
	})
}

func (pa *PlanAbsence) UnmarshalJSON(data []byte) error {
	var res planAbsence
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	*pa = PlanAbsence{
		Duration: time.Duration(res.Duration) * time.Second,
		Soc:      res.Soc,
	}

	return nil
}

// Validate validates the absence
func (pa *PlanAbsence) Validate() error {
	if pa == nil {
		return nil
	}
	if pa.Soc <= 0 || pa.Soc > 100 {
		return fmt.Errorf("absence soc drop out of range: %d", pa.Soc)
	}
	if pa.Duration <= 0 {
		return errors.New("absence duration must be positive")
	}
	return nil
}

type PlanStrategy struct {
	Continuous   bool          `json:"continuous"`   // force continuous planning
	Precondition time.Duration `json:"precondition"` // precondition duration in seconds
}

type planStrategy struct {
	Continuous   bool  `json:"continuous"`   // force continuous planning
	Precondition int64 `json:"precondition"` // precondition duration in seconds
}

func (ps PlanStrategy) MarshalJSON() ([]byte, error) {
	return json.Marshal(planStrategy{
		Continuous:   ps.Continuous,
		Precondition: int64(ps.Precondition.Seconds()),
	})
}

func (ps *PlanStrategy) UnmarshalJSON(data []byte) error {
	var res planStrategy
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	*ps = PlanStrategy{
		Continuous:   res.Continuous,
		Precondition: time.Duration(res.Precondition) * time.Second,
	}

	return nil
}
