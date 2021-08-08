package id

import (
	"strings"
	"time"
)

// Timestamp implements JSON unmarshal for RFC3339 string timestamp
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes RFC3339 string timestamp into time.Time
func (ct *Timestamp) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")

	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		(*ct).Time = t
	}

	return err
}
