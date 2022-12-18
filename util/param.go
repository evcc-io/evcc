package util

import (
	"strconv"
	"strings"
)

// Param is the broadcast channel data type
type Param struct {
	LoadPoint *int
	Sub       *ParamSub
	Key       string
	Val       interface{}
}

type ParamSub struct {
	SubKey string
	Index  int
}

// UniqueID returns unique identifier for parameter LoadPoint/Key combination
func (p Param) UniqueID() string {
	var b strings.Builder

	if p.LoadPoint != nil {
		b.WriteString(strconv.Itoa(*p.LoadPoint) + ".")
	}

	if p.Sub != nil {
		b.WriteString(p.Sub.SubKey + "." + strconv.Itoa(p.Sub.Index) + ".")
	}

	b.WriteString(p.Key)

	return b.String()
}
