package util

import (
	"fmt"
	"strconv"
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	if v < 0 {
		return 0, fmt.Errorf("invalid duration: %d", v)
	}

	return time.Duration(v) * time.Second, nil
}
