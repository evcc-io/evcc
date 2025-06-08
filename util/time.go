package util

import (
	"fmt"
	"time"
)

// GetNextOccurrence returns the next occurrence of the given time on the specified weekdays, as of a certain comparing timestamp.
func GetNextOccurrence(weekdays []int, timeStr string, tz string, compTime time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone: %w", err)
	}

	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format, expected HH:MM: %w", err)
	}

	hour, minute := parsedTime.Hour(), parsedTime.Minute()

	target := time.Date(
		compTime.Year(), compTime.Month(), compTime.Day(),
		hour, minute, 0, 0, loc,
	)

	// If the target time has passed today, start from tomorrow
	if !target.After(compTime) {
		target = target.AddDate(0, 0, 1)
	}

	// Check the next 7 days for a valid match
	for range 7 {
		weekday := int(target.Weekday())
		if contains(weekdays, weekday) {
			return target, nil
		}
		target = target.AddDate(0, 0, 1)
	}

	return time.Time{}, fmt.Errorf("no valid weekday found")
}

// helper function to check if a slice contains a value
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
