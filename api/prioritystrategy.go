package api

// PriorityStrategy determines how a loadpoint is ranked against other loadpoints
// of the same priority when distributing surplus power.
type PriorityStrategy int

//go:generate go tool enumer -type PriorityStrategy -trimprefix Priority -transform=lower -text
const (
	PriorityNone    PriorityStrategy = iota // no sub-ordering (default)
	PrioritySoc                             // prefer the lower vehicle soc
	PriorityDeficit                         // prefer the larger gap to limit soc
)

// PriorityBasis determines whether a priority strategy ranks loadpoints by soc
// percentage or by absolute energy (kWh).
type PriorityBasis int

//go:generate go tool enumer -type PriorityBasis -trimprefix PriorityBasis -transform=lower -text
const (
	PriorityBasisPercent PriorityBasis = iota // rank by soc-% (default)
	PriorityBasisEnergy                       // rank by absolute energy (kWh)
)
