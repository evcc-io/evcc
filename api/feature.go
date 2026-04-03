package api

type Feature int

//go:generate go tool enumer -type Feature -text
const (
	_                Feature = iota
	CoarseCurrent            // charger
	IntegratedDevice         // charger
	SwitchDevice             // charger - heat pumps or switch sockets
	Heating                  // charger - heating device
	Continuous               // charger - heating device where disabled means "normal operation"
	Average                  // tariff
	Cacheable                // tariff
	Offline                  // vehicle
	Retryable                // vehicle
	Streaming                // vehicle
	WelcomeCharge            // vehicle
)
