package util

import "time"

func timeRound(d time.Duration, round string) time.Duration {
	switch round {
	case "s", "sec":
		return d.Round(time.Second)
	case "m", "min":
		return d.Round(time.Minute)
	default:
		return d
	}
}
