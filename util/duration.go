package util

import (
	"strconv"
	"time"
)

// ParseDuration parses a string as integer seconds value and returns a time.Duration
func ParseDuration(s string) (time.Duration, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	return time.Duration(v) * time.Second, nil
}
