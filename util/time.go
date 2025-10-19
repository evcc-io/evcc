package util

import (
	"fmt"
	"slices"
	"time"
)

// GetNextOccurrence returns the next occurrence of the given time on the specified weekdays.
func GetNextOccurrence(now time.Time, weekdays []int, timeStr string, tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone: %w", err)
	}

	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format, expected HH:MM: %w", err)
	}

	hour, minute := parsedTime.Hour(), parsedTime.Minute()

	now = now.In(loc)

	target := time.Date(
		now.Year(), now.Month(), now.Day(),
		hour, minute, 0, 0, loc,
	)

	fmt.Println("GetNextOccurrence", "target 1", target)

	// If the target time has passed today, start from tomorrow
	if target.Before(now) {
		target = target.AddDate(0, 0, 1)
	}

	// Check the next 7 days for a valid match
	for range 7 {
		weekday := int(target.Weekday())
		if slices.Contains(weekdays, weekday) {
			fmt.Println("GetNextOccurrence", "target 2", target)
			return target, nil
		}
		target = target.AddDate(0, 0, 1)
	}

	return time.Time{}, fmt.Errorf("no valid weekday found")
}
