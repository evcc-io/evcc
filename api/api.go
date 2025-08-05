package api

import (
	"context"
	"io"
	"net/url"
	"time"
)

//go:generate go tool mockgen -package api -destination mock.go github.com/evcc-io/evcc/api Charger,ChargeState,CurrentLimiter,CurrentGetter,PhaseSwitcher,PhaseGetter,FeatureDescriber,Identifier,Meter,MeterEnergy,PhaseCurrents,Vehicle,ChargeRater,Battery,Tariff,BatteryController,Circuit

// Meter provides total active power in W
type Meter interface {
	CurrentPower() (float64, error)
}

// MeterEnergy provides total energy in kWh
type MeterEnergy interface {
	TotalEnergy() (float64, error)
}

// PhaseCurrents provides per-phase current A
type PhaseCurrents interface {
	Currents() (float64, float64, float64, error)
}

// PhaseVoltages provides per-phase voltage V
type PhaseVoltages interface {
	Voltages() (float64, float64, float64, error)
}

// PhasePowers provides signed per-phase power W
type PhasePowers interface {
	Powers() (float64, float64, float64, error)
}

// Battery provides battery Soc in %
type Battery interface {
	Soc() (float64, error)
}

// BatteryCapacity provides a capacity in kWh
type BatteryCapacity interface {
	Capacity() float64
}

// MaxACPowerGetter provides max AC power in W
type MaxACPowerGetter interface {
	MaxACPower() float64
}

// ChargeState provides current charging status
type ChargeState interface {
	Status() (ChargeStatus, error)
}

type StatusReasoner interface {
	StatusReason() (Reason, error)
}

// CurrentController provides settings charging maximum charging current
type CurrentController interface {
	MaxCurrent(current int64) error
}

// CurrentGetter provides getting charging maximum charging current for validation
type CurrentGetter interface {
	GetMaxCurrent() (float64, error)
}

// BatteryController optionally allows to control home battery (dis)charging behavior
type BatteryController interface {
	SetBatteryMode(BatteryMode) error
}

// Charger provides current charging status and enable/disable charging
type Charger interface {
	ChargeState
	Enabled() (bool, error)
	Enable(enable bool) error
	CurrentController
}

// ChargerEx provides milli-amp precision charger current control
type ChargerEx interface {
	MaxCurrentMillis(current float64) error
}

// PhaseSwitcher provides 1p3p switching
type PhaseSwitcher interface {
	Phases1p3p(phases int) error
}

type PhaseGetter interface {
	GetPhases() (int, error)
}

// Diagnosis is a helper interface that allows to dump diagnostic data to console
type Diagnosis interface {
	Diagnose()
}

// ChargeTimer provides current charge cycle duration
type ChargeTimer interface {
	ChargeDuration() (time.Duration, error)
}

// ChargeRater provides charged energy amount in kWh
type ChargeRater interface {
	ChargedEnergy() (float64, error)
}

// Identifier identifies a vehicle and is implemented by the charger
type Identifier interface {
	Identify() (string, error)
}

// Authorizer authorizes a charging session by supplying RFID credentials
type Authorizer interface {
	Authorize(key string) error
}

// PhaseDescriber returns the number of physically connected phases
// Used for vehicles and to limit switch sockets to 1p only
type PhaseDescriber interface {
	Phases() int
}

// Vehicle represents the EV and it's battery
type Vehicle interface {
	Battery
	BatteryCapacity
	IconDescriber
	FeatureDescriber
	PhaseDescriber
	TitleDescriber
	SetTitle(string)
	Identifiers() []string
	OnIdentified() ActionConfig
}

// VehicleFinishTimer provides estimated charge cycle finish time.
// Finish time is normalized for charging to 100% and may deviate from vehicle display if soc limit is effective.
type VehicleFinishTimer interface {
	FinishTime() (time.Time, error)
}

// VehicleRange provides the vehicles remaining km range
type VehicleRange interface {
	Range() (int64, error)
}

// VehicleClimater provides climatisation data
type VehicleClimater interface {
	Climater() (bool, error)
}

// VehicleOdometer returns the vehicles milage
type VehicleOdometer interface {
	Odometer() (float64, error)
}

// VehiclePosition returns the vehicles position in latitude and longitude
type VehiclePosition interface {
	Position() (float64, float64, error)
}

// CurrentLimiter returns the current limits
type CurrentLimiter interface {
	GetMinMaxCurrent() (float64, float64, error)
}

// SocLimiter returns the soc limit
type SocLimiter interface {
	GetLimitSoc() (int64, error)
}

// ChargeController allows to start/stop the charging session on the vehicle side
type ChargeController interface {
	ChargeEnable(bool) error
}

// Resurrector provides wakeup calls to the vehicle with an API call or a CP interrupt from the charger
type Resurrector interface {
	WakeUp() error
}

// Tariff is a tariff capable of retrieving tariff rates
type Tariff interface {
	Rates() (Rates, error)
	Type() TariffType
}

// AuthProvider is the ability to provide OAuth authentication through the ui
type AuthProvider interface {
	Login(state string) string
	Logout() error
	HandleCallback(responseValues url.Values) error
	Authenticated() bool
	DisplayName() string
}

// IconDescriber optionally provides an icon
type IconDescriber interface {
	Icon() string
}

// FeatureDescriber optionally provides a list of supported non-api features
type FeatureDescriber interface {
	Features() []Feature
}

// TitleDescriber optionally provides an title
type TitleDescriber interface {
	GetTitle() string
}

// CsvWriter converts to csv
type CsvWriter interface {
	WriteCsv(context.Context, io.Writer) error
}

// CircuitMeasurements is the measurements a circuit or load must deliver
type CircuitMeasurements interface {
	GetChargePower() float64
	GetMaxPhaseCurrent() float64
}

// CircuitLoad represents a loadpoint attached to a circuit
type CircuitLoad interface {
	CircuitMeasurements
	GetCircuit() Circuit
}

// Circuit defines the load control domain
type Circuit interface {
	CircuitMeasurements
	GetTitle() string
	SetTitle(string)
	GetParent() Circuit
	RegisterChild(child Circuit)
	Wrap(parent Circuit) error
	HasMeter() bool
	GetMaxPower() float64
	GetMaxCurrent() float64
	SetMaxPower(float64)
	SetMaxCurrent(float64)
	Update([]CircuitLoad) error
	ValidateCurrent(old, new float64) float64
	ValidatePower(old, new float64) float64
}

// Redactor is an interface to redact sensitive data
type Redactor interface {
	Redacted() any
}
