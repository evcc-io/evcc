package util

import (
	"strconv"
	"strings"
)

// Param is the broadcast channel data type
type Param struct {
	Loadpoint *int
	Circuit   *string
	Key       string
	Val       interface{}
}

// UniqueID returns unique identifier for parameter Loadpoint/Key combination
func (p Param) UniqueID() string {

	var b strings.Builder

	if p.Loadpoint != nil {
		b.WriteString(strconv.Itoa(*p.Loadpoint) + ".")
	} else if p.Circuit != nil {
		b.WriteString(*p.Circuit + ".")
	}

	b.WriteString(p.Key)

	return b.String()
}
