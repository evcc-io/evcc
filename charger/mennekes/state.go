//go:generate stringer -type=EVSEState -output=./state_string.go

package mennekes

type EVSEState uint16

const (
	NotInitialized     EVSEState = iota
	Idle                         // A1
	EVConnected                  // B1
	PreconditionsValid           // B2
	ReadyToCharge                // C2
	Charging                     // D2
	Error                        // E2
	ServiceMode                  // F2
)
