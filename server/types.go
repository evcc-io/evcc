package server

import (
	"time"
)

type planGoal[T any] struct {
	Time  time.Time `json:"time"`
	Value T         `json:"value"`
}
