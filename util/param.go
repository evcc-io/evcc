package util

import "strconv"

// Param is the broadcast channel data type
type Param struct {
	LoadPoint *int
	Key       string
	Val       interface{}
}

// UniqueID returns unique identifier for parameter LoadPoint/Key combination
func (p Param) UniqueID() string {
	key := p.Key
	if p.LoadPoint != nil {
		key = strconv.Itoa(*p.LoadPoint) + "." + key
	}
	return key
}
