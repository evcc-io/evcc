package api

type RepeatingPlanStruct struct {
	Weekdays     []int  `json:"weekdays"`     // 0-6 (Sunday-Saturday)
	Time         string `json:"time"`         // HH:MM
	Tz           string `json:"tz"`           // timezone in IANA format
	Soc          int    `json:"soc"`          // target soc
	Precondition int64  `json:"precondition"` // precondition duration in seconds
	Active       bool   `json:"active"`       // active flag
}
