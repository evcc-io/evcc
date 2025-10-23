package api

import "time"

type RepeatingPlanStruct struct {
	Weekdays      []int  `json:"weekdays"`               // 0-6 (Sunday-Saturday)
	Time          string `json:"time"`                   // HH:MM
	Tz            string `json:"tz"`                     // timezone in IANA format
	Soc           int    `json:"soc"`                    // target soc
	Precondition_ *int64 `json:"precondition,omitempty"` // deprecated
	Active        bool   `json:"active"`                 // active flag
}

type PlanStrategy struct {
	Continuous   bool          `json:"continuous"`   // force continuous planning
	Precondition time.Duration `json:"precondition"` // precondition duration in seconds
}
