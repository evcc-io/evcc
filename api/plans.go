package api

import (
	"encoding/json"
	"time"
)

type RepeatingPlan struct {
	Weekdays []int  `json:"weekdays"` // 0-6 (Sunday-Saturday)
	Time     string `json:"time"`     // HH:MM
	Tz       string `json:"tz"`       // timezone in IANA format
	Soc      int    `json:"soc"`      // target soc
	Active   bool   `json:"active"`   // active flag
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
