package modbus

import "strings"

// ReadingName formats MBMD reading names
func ReadingName(val string) string {
	if len(val) > 0 {
		val = strings.ToUpper(val[:1]) + val[1:]
	}
	return val
}
