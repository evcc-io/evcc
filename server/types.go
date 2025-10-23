package server

type planStrategyPayload struct {
	Continuous   bool  `json:"continuous"`
	Precondition int64 `json:"precondition"`
}
