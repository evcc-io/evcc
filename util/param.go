package util

import "strconv"

// Param is the broadcast channel data type
type Param struct {
	Loadpoint *int
	Key       string
	Val       interface{}
}

// UniqueID returns unique identifier for parameter Loadpoint/Key combination
func (p Param) UniqueID() string {
	key := p.Key
	if p.Loadpoint != nil {
		key = strconv.Itoa(*p.Loadpoint) + "." + key
	}
	return key
}
