package api

type Feature int

//go:generate enumer -type Feature -text
const (
	_ Feature = iota
	Offline
	CoarseCurrent
	IntegratedDevice
	Heating
	Retryable
)
