package modbus

import (
	"fmt"
	"strings"
)

type ReadOnlyMode int

const (
	ReadOnlyPassthrough ReadOnlyMode = iota
	ReadOnlyDeny
	ReadOnly
)

// ReadOnlyModeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func ReadOnlyModeString(s string) (ReadOnlyMode, error) {
	switch l := strings.ToLower(s); l {
	case "", "false":
		return ReadOnlyPassthrough, nil
	case "deny":
		return ReadOnlyDeny, nil
	case "ignore", "true":
		return ReadOnly, nil
	default:
		return 0, fmt.Errorf("%s does not belong to ReadOnlyMode values", l)
	}
}
