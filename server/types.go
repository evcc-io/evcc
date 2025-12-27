package server

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

type planStrategyPayload struct {
	Continuous   bool  `json:"continuous"`
	Precondition int64 `json:"precondition"`
}

func planStrategyPayloadFromApi(ps api.PlanStrategy) planStrategyPayload {
	return planStrategyPayload{
		Continuous:   ps.Continuous,
		Precondition: int64(ps.Precondition.Seconds()),
	}
}

type planGoal[T any] struct {
	Time  time.Time `json:"time"`
	Value T         `json:"value"`
}
