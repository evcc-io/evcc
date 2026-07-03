package fritz

// Meter defines the interface for Fritz connections (both legacy LUA and REST)
type Meter interface {
	CurrentPower() (float64, error)
	TotalEnergy() (float64, error)
}

// Switch extends Meter with switch control capabilities
type Switch interface {
	Meter
	SwitchPresent() (bool, error)
	SwitchState() (bool, error)
	SwitchOn() error
	SwitchOff() error
}
