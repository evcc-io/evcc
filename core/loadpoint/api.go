package loadpoint

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// Controller gives access to loadpoint
type Controller interface {
	LoadpointControl(API)
}

// API is the external loadpoint API
type API interface {
	// Name returns the defined loadpoint name
	Name() string
	// HasChargeMeter determines if a physical charge meter is attached
	HasChargeMeter() bool

	//
	// status
	//

	// GetStatus returns the charging status
	GetStatus() api.ChargeStatus

	//
	// settings
	//

	// GetMode returns loadpoint charge mode
	GetMode() api.ChargeMode
	// SetMode sets loadpoint charge mode
	SetMode(api.ChargeMode)
	// GetTargetSoC returns loadpoint charge target soc
	GetTargetSoC() int
	// SetTargetSoC sets loadpoint charge target soc
	SetTargetSoC(int) error
	// GetMinSoC returns loadpoint charge minimum soc
	GetMinSoC() int
	// SetMinSoC sets loadpoint charge minimum soc
	SetMinSoC(int) error
	// GetPhases returns loadpoint enabled phases
	GetPhases() int
	// SetPhases sets loadpoint enabled phases
	SetPhases(int) error

	// SetTargetCharge sets loadpoint charge targetSoC
	SetTargetCharge(time.Time, int)
	// RemoteControl sets remote status demand
	RemoteControl(string, RemoteDemand)

	//
	// power and energy
	//

	// GetChargePower returns the current charge power
	GetChargePower() float64
	// GetMinCurrent returns the min loadpoint current
	GetMinCurrent() float64
	// SetMinCurrent returns the min loadpoint current
	SetMinCurrent(float64)
	// GetMaxCurrent returns the max loadpoint current
	GetMaxCurrent() float64
	// SetMaxCurrent returns the max loadpoint current
	SetMaxCurrent(float64)
	// GetMinPower returns the min loadpoint power for a single phase
	GetMinPower() float64
	// GetMaxPower returns the max loadpoint power taking active phases into account
	GetMaxPower() float64

	//
	// charge progress
	//

	// GetRemainingDuration is the estimated remaining charging duration
	GetRemainingDuration() time.Duration
	// GetRemainingEnergy is the remaining charge energy in Wh
	GetRemainingEnergy() float64
}
