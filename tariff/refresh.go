package tariff

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// nextRefresh returns the time until the next price update
func (t *embed) nextRefresh(interval time.Duration, data *util.Monitor[api.Rates]) time.Duration {
	var rates api.Rates
	_ = data.GetFunc(func(val api.Rates) {
		rates = val
	})

	return refreshDelay(time.Now(), interval, t.EarlyRefresh, rates)
}

// refreshDelay returns the time until the next price update.
// Day-ahead prices are usually published around 13:00 local time. With early refresh
// enabled and tomorrow's prices still missing, prices are refreshed at the early interval
// between 13:00 and 15:00 and every 30 minutes after that.
func refreshDelay(now time.Time, interval, early time.Duration, rates api.Rates) time.Duration {
	if early <= 0 {
		return interval
	}

	local := now.Local()
	date := func(day, hour int) time.Time {
		return time.Date(local.Year(), local.Month(), local.Day()+day, hour, 0, 0, 0, time.Local)
	}

	endOfTomorrow := date(2, 0)
	for _, r := range rates {
		if !r.End.Before(endOfTomorrow) {
			return interval
		}
	}

	switch publish, late := date(0, 13), date(0, 15); {
	case local.Before(publish):
		return min(interval, publish.Sub(local))
	case local.Before(late):
		return min(interval, early)
	default:
		return min(interval, 30*time.Minute)
	}
}
