package api

import "time"

type RepeatingPlan struct {
	Weekdays     []int  `json:"weekdays"`     // 0-6 (Sunday-Saturday)
	Time         string `json:"time"`         // HH:MM
	Tz           string `json:"tz"`           // timezone in IANA format
	Soc          int    `json:"soc"`          // target soc
	Active       bool   `json:"active"`       // active flag
	Precondition int64  `json:"-" todo:"..."` // TODO deprecated
}

type PlanStrategy struct {
	Continuous   bool          `json:"continuous"`   // force continuous planning
	Precondition time.Duration `json:"precondition"` // precondition duration in seconds
}
