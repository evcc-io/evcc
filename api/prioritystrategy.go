package api

import (
	"encoding"
	"fmt"
	"strings"
)

// PriorityStrategy determines how a loadpoint is ranked against other loadpoints
// of the same priority when distributing surplus power. Valid values are static,
// soc and deficit.
type PriorityStrategy string

// Priority strategies
const (
	// PriorityStatic ranks loadpoints by their configured priority only (default).
	PriorityStatic PriorityStrategy = ""
	// PrioritySoc additionally prefers the loadpoint with the lower vehicle soc
	// among loadpoints of the same priority.
	PrioritySoc PriorityStrategy = "soc"
	// PriorityDeficit additionally prefers the loadpoint with the larger gap
	// between vehicle soc and its limit soc among loadpoints of the same priority.
	PriorityDeficit PriorityStrategy = "deficit"
)

// PriorityStrategyString converts a string to PriorityStrategy
func PriorityStrategyString(s string) (PriorityStrategy, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "static":
		return PriorityStatic, nil
	case string(PrioritySoc):
		return PrioritySoc, nil
	case string(PriorityDeficit):
		return PriorityDeficit, nil
	default:
		return PriorityStatic, fmt.Errorf("invalid priority strategy: %s", s)
	}
}

var _ encoding.TextUnmarshaler = (*PriorityStrategy)(nil)

func (s *PriorityStrategy) UnmarshalText(text []byte) error {
	var err error
	*s, err = PriorityStrategyString(string(text))
	return err
}

// PriorityBasis determines whether a priority strategy ranks loadpoints by soc
// percentage or by absolute energy (kWh). Valid values are percent and energy.
type PriorityBasis string

// Priority bases
const (
	// PriorityBasisPercent ranks the priority strategy in soc-% (default), so a
	// percentage gap is compared regardless of battery capacity.
	PriorityBasisPercent PriorityBasis = ""
	// PriorityBasisEnergy ranks the priority strategy by absolute energy (kWh) by
	// scaling the soc-% gap with the vehicle capacity, so a smaller battery is not
	// over-prioritized just because its percentage is lower. Falls back to percent
	// when the vehicle capacity is unknown.
	PriorityBasisEnergy PriorityBasis = "energy"
)

// PriorityBasisString converts a string to PriorityBasis
func PriorityBasisString(s string) (PriorityBasis, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "percent", "percentage", "soc":
		return PriorityBasisPercent, nil
	case string(PriorityBasisEnergy), "absolute", "kwh":
		return PriorityBasisEnergy, nil
	default:
		return PriorityBasisPercent, fmt.Errorf("invalid priority basis: %s", s)
	}
}

var _ encoding.TextUnmarshaler = (*PriorityBasis)(nil)

func (b *PriorityBasis) UnmarshalText(text []byte) error {
	var err error
	*b, err = PriorityBasisString(string(text))
	return err
}
