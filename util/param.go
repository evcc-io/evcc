package util

import (
	"strconv"
	"strings"
)

// Param is the broadcast channel data type
type Param struct {
	LoadPoint *int
	Key       string
	Val       interface{}
}

// UniqueID returns unique identifier for parameter LoadPoint/Key combination
func (p Param) UniqueID() string {
	var b strings.Builder

	if p.LoadPoint != nil {
		b.WriteString(strconv.Itoa(*p.LoadPoint) + ".")
	}

	b.WriteString(p.Key)

	return b.String()
}
