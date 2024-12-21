package api

type Feature int

//go:generate go tool enumer -type Feature -text
const (
	_ Feature = iota
	Offline
	CoarseCurrent
	IntegratedDevice
	Heating
	Retryable
	WelcomeCharge
)
