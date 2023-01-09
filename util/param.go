package util

import (
	"strconv"
	"strings"
)

// Param is the broadcast channel data type
type Param struct {
	Loadpoint *int
	Key       string
	Val       interface{}
}

// UniqueID returns unique identifier for parameter Loadpoint/Key combination
func (p Param) UniqueID() string {
	var b strings.Builder

	if p.Loadpoint != nil {
		b.WriteString(strconv.Itoa(*p.Loadpoint) + ".")
	}

	b.WriteString(p.Key)

	return b.String()
}
