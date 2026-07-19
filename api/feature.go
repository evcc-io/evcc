package api

type Feature int

//go:generate go tool enumer -type Feature -text
const (
	_                        Feature = iota
	CoarseCurrent                    // charger
	IntegratedDevice                 // charger - always connected - no vehicle, no charging sessions
	SwitchDevice                     // charger - no current control - heat pumps or switch sockets
	Heating                          // charger - heating device - soc ist temperature (°C)
	DemandProfileWeekly              // charger - demand forecast: same weekday last week (warm water)
	DemandProfileTemperature         // charger - demand forecast: daily avg scaled by outdoor temp (room heating)
	Continuous                       // charger - heating device where disabled means "normal operation"
	Average                          // tariff
	Cacheable                        // tariff
	Offline                          // vehicle
	Retryable                        // vehicle
	Streaming                        // vehicle
	WelcomeCharge                    // vehicle
	ClimaterDisabled                 // vehicle - ignore climater state for charge control
	AutodetectDisabled               // vehicle - do not try to identify vehicle by status
	WakeUpDisabled                   // vehicle - do not send wake-up calls
)
