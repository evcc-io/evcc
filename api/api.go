package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//go:generate mockgen -package mock -destination ../mock/mock_api.go github.com/evcc-io/evcc/api Charger,ChargeState,PhaseSwitcher,Identifier,Meter,MeterEnergy,Vehicle,ChargeRater,Battery,Tariff

// ChargeMode is the charge operation mode. Valid values are off, now, minpv and pv
type ChargeMode string

// Charge modes
const (
	ModeEmpty ChargeMode = ""
	ModeOff   ChargeMode = "off"
	ModeNow   ChargeMode = "now"
	ModeMinPV ChargeMode = "minpv"
	ModePV    ChargeMode = "pv"
)

// String implements Stringer
func (c ChargeMode) String() string {
	return string(c)
}

// ChargeStatus is the EV's charging status from A to F
type ChargeStatus string

// Charging states
const (
	StatusNone ChargeStatus = ""
	StatusA    ChargeStatus = "A" // Fzg. angeschlossen: nein    Laden aktiv: nein    - Kabel nicht angeschlossen
	StatusB    ChargeStatus = "B" // Fzg. angeschlossen:   ja    Laden aktiv: nein    - Kabel angeschlossen
	StatusC    ChargeStatus = "C" // Fzg. angeschlossen:   ja    Laden aktiv:   ja    - Laden
	StatusD    ChargeStatus = "D" // Fzg. angeschlossen:   ja    Laden aktiv:   ja    - Laden mit Lüfter
	StatusE    ChargeStatus = "E" // Fzg. angeschlossen:   ja    Laden aktiv: nein    - Fehler (Kurzschluss)
	StatusF    ChargeStatus = "F" // Fzg. angeschlossen:   ja    Laden aktiv: nein    - Fehler (Ausfall Wallbox)
)

// ChargeStatusString converts a string to ChargeStatus
func ChargeStatusString(s string) (ChargeStatus, error) {
	switch status := strings.ToUpper(s); status {
	case "A", "B", "C":
		return ChargeStatus(status), nil
	case "D", "E", "F":
		return ChargeStatus(status), fmt.Errorf("invalid status: %s", status)
	default:
		return StatusNone, fmt.Errorf("invalid status: %s", s)
	}
}

// ChargeStatusStringWithMapping converts a string to ChargeStatus. In case of error, mapping is applied.
func ChargeStatusStringWithMapping(s string, m map[ChargeStatus]ChargeStatus) (ChargeStatus, error) {
	status, err := ChargeStatusString(s)
	if mappedStatus, ok := m[status]; ok && err != nil {
		return mappedStatus, nil
	}
	return status, err
}

// String implements Stringer
func (c ChargeStatus) String() string {
	return string(c)
}

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

// ChargeState provides current charging status
type ChargeState interface {
	Status() (ChargeStatus, error)
}

// CurrentLimiter provides settings charging maximum charging current
type CurrentLimiter interface {
	MaxCurrent(current int64) error
}

// CurrentGetter provides getting charging maximum charging current for validation
type CurrentGetter interface {
	GetMaxCurrent() (float64, error)
}

// Charger provides current charging status and enable/disable charging
type Charger interface {
	ChargeState
	Enabled() (bool, error)
	Enable(enable bool) error
	CurrentLimiter
}

// ChargerEx provides milli-amp precision charger current control
type ChargerEx interface {
	MaxCurrentMillis(current float64) error
}

// PhaseSwitcher provides 1p3p switching
type PhaseSwitcher interface {
	Phases1p3p(phases int) error
}

// Diagnosis is a helper interface that allows to dump diagnostic data to console
type Diagnosis interface {
	Diagnose()
}

// ChargeTimer provides current charge cycle duration
type ChargeTimer interface {
	ChargingTime() (time.Duration, error)
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

// Vehicle represents the EV and it's battery
type Vehicle interface {
	Battery
	BatteryCapacity
	IconDescriber
	Title() string
	SetTitle(string)
	Phases() int
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

// SocLimiter returns the vehicles charge limit
type SocLimiter interface {
	TargetSoc() (float64, error)
}

// VehicleChargeController allows to start/stop the charging session on the vehicle side
type VehicleChargeController interface {
	StartCharge() error
	StopCharge() error
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
	SetCallbackParams(baseURL, redirectURL string, authenticated chan<- bool)
	LoginHandler() http.HandlerFunc
	LogoutHandler() http.HandlerFunc
}

// IconDescriber optionally provides an icon
type IconDescriber interface {
	Icon() string
}

// FeatureDescriber optionally provides a list of supported non-api features
type FeatureDescriber interface {
	Features() []Feature
}

// CsvWriter converts to csv
type CsvWriter interface {
	WriteCsv(context.Context, io.Writer) error
}
