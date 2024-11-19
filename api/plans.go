package api

type RepeatingPlanStruct struct {
	Weekdays []int  `json:"weekdays"`
	Time     string `json:"time"`
	Soc      int    `json:"soc"`
	Active   bool   `json:"active"`
}
