package util

import (
	"time"
)

// GetNextOccurrence returns the next occurrence of the given time on the specified weekdays.
func GetNextOccurrence(weekdays []int, timeStr string) (time.Time, error) {
	now := time.Now()

	planTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, err
	}

	var nextOccurrence time.Time
	for _, w := range weekdays {
		// Calculate the difference in days to the target weekday
		dayOffset := (w - int(now.Weekday()) + 7) % 7

		// If the user has selected the day of the week that is today, and at the same time the user
		// has selected a time that would be in the past for today, the next day of the week in a week should be used
		if dayOffset == 0 && (now.Hour()*60+now.Minute()) > (planTime.Hour()*60+planTime.Minute()) {
			dayOffset = 7
		}

		// Adjust the current timestamp to the target weekday and set the time
		timestamp := now.AddDate(0, 0, dayOffset).Truncate(24 * time.Hour).Add(time.Hour*time.Duration(planTime.Hour()) + time.Minute*time.Duration(planTime.Minute()))

		// If this is the first occurrence or an earlier occurrence, update nextOccurrence
		if nextOccurrence.IsZero() || timestamp.Before(nextOccurrence) {
			nextOccurrence = timestamp
		}
	}

	return nextOccurrence, nil
}
