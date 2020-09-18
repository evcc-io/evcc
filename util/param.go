package util

import "strconv"

// Param is the broadcast channel data type
type Param struct {
	LoadPoint *int
	Key       string
	Val       interface{}
}

// UniqueID returns unique identifier for parameter LoadPoint/Key combination
func (p Param) UniqueID(key string, loadpoint *int) string {
	if loadpoint != nil {
		key = strconv.Itoa(*loadpoint) + "." + key
	}
	return key
}

// IsNil returns unique identifier for parameter LoadPoint/Key combination
func (p Param) IsNil() bool {
	return p.Key == ""
}
