package globalconfig

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (c *FTPBackup) Validate() error {
	schedule := strings.TrimSpace(c.Schedule)
	if schedule != "" {
		parts := strings.Split(schedule, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid schedule %q: expected format HH:MM", c.Schedule)
		}

		hour, err := strconv.Atoi(parts[0])
		if err != nil || hour < 0 || hour > 23 {
			return fmt.Errorf("invalid schedule %q: hour must be between 00 and 23", c.Schedule)
		}

		minute, err := strconv.Atoi(parts[1])
		if err != nil || minute < 0 || minute > 59 {
			return fmt.Errorf("invalid schedule %q: minute must be between 00 and 59", c.Schedule)
		}
	}

	timeout := strings.TrimSpace(c.Timeout)
	if timeout != "" {
		duration, err := time.ParseDuration(timeout)
		if err != nil || duration <= 0 {
			return fmt.Errorf("invalid timeout %q: expected a positive duration", c.Timeout)
		}
	}

	return nil
}
