package api

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/fatih/structs"
)

//go:generate mockgen -package mock -destination ../mock/mock_api.go github.com/evcc-io/evcc/api Charger,ChargeState,PhaseSwitcher,Identifier,Meter,MeterEnergy,Vehicle,ChargeRater,Battery

// ChargeMode are charge modes modeled after OpenWB
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

// String implements Stringer
func (c ChargeStatus) String() string {
	return string(c)
}

// ActionConfig defines an action to take on event
type ActionConfig struct {
	Mode       *ChargeMode `mapstructure:"mode,omitempty"`       // Charge Mode
	MinCurrent *float64    `mapstructure:"minCurrent,omitempty"` // Minimum Current
	MaxCurrent *float64    `mapstructure:"maxCurrent,omitempty"` // Maximum Current
	MinSoC     *int        `mapstructure:"minSoC,omitempty"`     // Minimum SoC
	TargetSoC  *int        `mapstructure:"targetSoC,omitempty"`  // Target SoC
}

// Merge merges all non-nil properties of the additional config into the base config.
// The receiver's config remains immutable.
func (a ActionConfig) Merge(m ActionConfig) ActionConfig {
	if m.Mode != nil {
		a.Mode = m.Mode
	}
	if m.MinCurrent != nil {
		a.MinCurrent = m.MinCurrent
	}
	if m.MaxCurrent != nil {
		a.MaxCurrent = m.MaxCurrent
	}
	if m.MinSoC != nil {
		a.MinSoC = m.MinSoC
	}
	if m.TargetSoC != nil {
		a.TargetSoC = m.TargetSoC
	}
	return a
}

// String implements Stringer and returns the ActionConfig as comma-separated key:value string
func (a ActionConfig) String() string {
	var s []string
	for k, v := range structs.Map(a) {
		val := reflect.ValueOf(v)
		if v != nil && !val.IsNil() {
			s = append(s, fmt.Sprintf("%s:%v", k, val.Elem()))
		}
	}
	return strings.Join(s, ", ")
}

// Meter is able to provide current power in W
type Meter interface {
	CurrentPower() (float64, error)
}

// MeterEnergy is able to provide current energy in kWh
type MeterEnergy interface {
	TotalEnergy() (float64, error)
}

// MeterCurrent is able to provide per-line current A
type MeterCurrent interface {
	Currents() (float64, float64, float64, error)
}

// Battery is able to provide battery SoC in %
type Battery interface {
	SoC() (float64, error)
}

// ChargeState provides current charging status
type ChargeState interface {
	Status() (ChargeStatus, error)
}

// Charger is able to provide current charging status and enable/disable charging
type Charger interface {
	ChargeState
	Enabled() (bool, error)
	Enable(enable bool) error
	MaxCurrent(current int64) error
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
	Title() string
	Capacity() int64
	Phases() int
	Identifiers() []string
	OnIdentified() ActionConfig
}

// VehicleFinishTimer provides estimated charge cycle finish time
type VehicleFinishTimer interface {
	FinishTime() (time.Time, error)
}

// VehicleRange provides the vehicles remaining km range
type VehicleRange interface {
	Range() (int64, error)
}

// VehicleClimater provides climatisation data
type VehicleClimater interface {
	Climater() (active bool, outsideTemp, targetTemp float64, err error)
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
	TargetSoC() (float64, error)
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

// Tariff is the grid tariff
type Tariff interface {
	IsCheap() (bool, error)
	CurrentPrice() (float64, error) // EUR/kWh, CHF/kWh, ...
}

// AuthProvider is the ability to provide OAuth authentication through the ui
type AuthProvider interface {
	SetCallbackParams(baseURL, redirectURL string, authenticated chan<- bool)
	LoginHandler() http.HandlerFunc
	LogoutHandler() http.HandlerFunc
}
