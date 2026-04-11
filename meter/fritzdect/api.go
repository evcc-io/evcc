package fritzdect

// API defines the interface for Fritz DECT connections (both legacy LUA and REST)
type API interface {
	CurrentPower() (float64, error)
	TotalEnergy() (float64, error)
}

// SwitchAPI extends API with switch control capabilities
type SwitchAPI interface {
	API
	SwitchPresent() (bool, error)
	SwitchState() (bool, error)
	SwitchOn() error
	SwitchOff() error
}
