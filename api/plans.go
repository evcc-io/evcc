package api

import (
	"time"
)

type PlanStruct struct {
	Id     int       `json:"Id"`
	Soc    int       `json:"soc"`
	Time   time.Time `json:"time"`
	Active bool      `json:"active"`
}

type RepeatingPlanStruct struct {
	Weekdays []int  `json:"weekdays"`
	Time     string `json:"time"`
	Soc      int    `json:"soc"`
	Active   bool   `json:"active"`
}

func (p *RepeatingPlanStruct) ToPlansWithTimestamp(id int) []PlanStruct {
	var formattedPlans []PlanStruct

	now := time.Now()

	planTime, err := time.Parse("15:04", p.Time)
	if err != nil {
		return []PlanStruct{}
	}

	for _, w := range p.Weekdays {
		// Calculate the difference in days to the target weekday
		dayOffset := (w - int(now.Weekday()) + 7) % 7

		// If the user has selected the day of the week that is today, and at the same time the user
		// has selected a time that would be in the past for today, the next day of the week in a week should be used
		if dayOffset == 0 && (now.UTC().Hour()*60+now.UTC().Minute()) > (planTime.Hour()*60+planTime.Minute()) {
			dayOffset = 7
		}

		// Adjust the current timestamp to the target weekday and set the time
		timestamp := now.AddDate(0, 0, dayOffset).Truncate(24 * time.Hour).Add(time.Hour*time.Duration(planTime.Hour()) + time.Minute*time.Duration(planTime.Minute()))

		// Append the resulting plan with the calculated timestamp
		formattedPlans = append(formattedPlans, PlanStruct{
			Id:     id,
			Soc:    p.Soc,
			Time:   timestamp,
			Active: p.Active,
		})
	}

	return formattedPlans
}
