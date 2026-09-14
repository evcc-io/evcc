package util

import (
	"fmt"
	"slices"
	"time"
)

// GetNextOccurrence returns the next occurrence of the given time on the specified weekdays starting from the current time.
func GetNextOccurrence(weekdays []int, timeStr string, tz string) (time.Time, error) {
	return GetNextOccurrenceAt(time.Now(), weekdays, timeStr, tz)
}

// GetNextOccurrenceAt returns the next occurrence of the given time on the specified weekdays starting from the provided reference time.
func GetNextOccurrenceAt(from time.Time, weekdays []int, timeStr string, tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone: %w", err)
	}

	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format, expected HH:MM: %w", err)
	}

	hour, minute := parsedTime.Hour(), parsedTime.Minute()

	from = from.In(loc)

	target := time.Date(
		from.Year(), from.Month(), from.Day(),
		hour, minute, 0, 0, loc,
	)

	// If the target time has passed today, start from tomorrow
	if target.Before(from) {
		target = target.AddDate(0, 0, 1)
	}

	// Check the next 7 days for a valid match
	for range 7 {
		weekday := int(target.Weekday())
		if slices.Contains(weekdays, weekday) {
			return target, nil
		}
		target = target.AddDate(0, 0, 1)
	}

	return time.Time{}, fmt.Errorf("no valid weekday found")
}
