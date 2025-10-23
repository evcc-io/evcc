package server

import "time"

type planStrategyPayload struct {
	Continuous   bool  `json:"continuous"`
	Precondition int64 `json:"precondition"`
}

type planGoal[T any] struct {
	Time  time.Time `json:"time"`
	Value T         `json:"value"`
}
