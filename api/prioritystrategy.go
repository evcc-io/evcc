package api

import (
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

// String implements Stringer
func (s PriorityStrategy) String() string {
	if s == PriorityStatic {
		return "static"
	}
	return string(s)
}

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
