//go:generate stringer -type=PhaseSwitchingMode,RequestedPhases -output=./phase_string.go

package mennekes

type PhaseSwitchingMode uint16

const (
	PhaseSolarOnly1Phase PhaseSwitchingMode = iota
	PhaseSolarOnly3Phases
	PhaseSolarDynamic1or3Phases
)

type RequestedPhases uint16

const (
	AllAvailablePhases RequestedPhases = 0
	Force1PhaseOnly    RequestedPhases = 1
)
