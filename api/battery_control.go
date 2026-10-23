package api

import "time"

// BatteryChargePowerUpdater changes power without re-entering manual mode.
// Callers must first establish and verify their own manual control.
type BatteryChargePowerUpdater interface {
	UpdateBatteryChargePower(power float64) error
}

// BatteryOperatingMode describes the actual device mode, not evcc's requested mode.
type BatteryOperatingMode string

const (
	BatteryOperatingUnknown BatteryOperatingMode = "unknown"
	BatteryOperatingAuto    BatteryOperatingMode = "automatic"
	BatteryOperatingManual  BatteryOperatingMode = "manual"
)

// BatteryControlState is a fresh device observation. Power is positive for discharge.
// LeaseID must come from an enforcing device/controller, never from local command history.
type BatteryControlState struct {
	Identity       string   // configured device identity, never a secret
	ChargeSetpoint *float64 // optional readback, not measured charging power
	Mode           BatteryOperatingMode
	NativeMode     string
	Power          float64
	Soc            float64
	NativeCharging bool
	ObservedAt     time.Time
	LeaseID        string
	LeaseUntil     time.Time
}

// BatteryControlRestorer restores an observed native automatic mode.
// An empty mode explicitly requests the configured automatic default.
type BatteryControlRestorer interface {
	RestoreBatteryMode(nativeMode string) error
}

// BatteryControlStateReader reports actual operating mode and battery measurements.
type BatteryControlStateReader interface {
	BatteryControlState() (BatteryControlState, error)
}

// BatteryPVLeaseController provides fenced, expiring control of PV charging.
// Acquire must atomically refuse manual mode, native charging and another owner.
// All writes are conditional on the lease ID; expiry independently stops forced
// charging and restores the prior automatic mode even if evcc crashes or loses power.
// An unfenced HTTP mode/setpoint sequence does not satisfy this contract.
type BatteryPVLeaseController interface {
	BatteryControlStateReader
	AcquireBatteryPVControl(id string, power float64, ttl time.Duration) error
	RenewBatteryPVControl(id string, power float64, ttl time.Duration) error
	ReleaseBatteryPVControl(id string) error
}
