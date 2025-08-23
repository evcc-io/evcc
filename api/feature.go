package api

type Feature int

//go:generate go tool enumer -type Feature -text
const (
	_                Feature = iota
	CoarseCurrent            // charger
	IntegratedDevice         // charger
	Heating                  // charger
	Cacheable                // tariff
	Offline                  // vehicle
	Retryable                // vehicle
	WelcomeCharge            // vehicle
)
