package chargercontroller

// Controller translates power targets into charger commands.
// The loadpoint calls SetOfferedPower() with a target in watts;
// the controller handles the specifics of how to deliver that power.
type Controller interface {
	// SetOfferedPower sets the target charging power in watts.
	// 0 or below MinPower means disable.
	SetOfferedPower(power float64) error

	// SetMaxPower requests maximum power output.
	// For current controllers, this forces max phases before setting max current.
	SetMaxPower() error

	// MinPower returns the minimum non-zero charging power.
	MinPower() float64

	// MaxPower returns the maximum charging power.
	MaxPower() float64

	// EffectiveChargePower returns the charge power adjusted for controller-specific behavior.
	// For current controllers, this accounts for vehicle current hysteresis.
	// For power controllers, this returns the measured charge power as-is.
	EffectiveChargePower() float64
}
