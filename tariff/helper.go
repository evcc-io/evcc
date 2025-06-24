package tariff

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jinzhu/now"
)

// Name returns the tariff type name
func Name(conf config.Typed) string {
	if conf.Other != nil && conf.Other["tariff"] != nil {
		return conf.Other["tariff"].(string)
	}
	return conf.Type
}

func bo() backoff.BackOff {
	return backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Second),
		backoff.WithMaxElapsedTime(time.Minute),
	)
}

// backoffPermanentError returns a permanent error in case of HTTP 400
func backoffPermanentError(err error) error {
	if se := new(request.StatusError); errors.As(err, &se) {
		if code := se.StatusCode(); code >= 400 && code <= 599 {
			return backoff.Permanent(se)
		}
	}
	if err != nil && strings.HasPrefix(err.Error(), "jq: query failed") {
		return backoff.Permanent(err)
	}
	return err
}

// mergeRates blends new and existing rates, keeping existing rates after current hour
func mergeRates(data *util.Monitor[api.Rates], new api.Rates) {
	mergeRatesAfter(data, new, now.With(time.Now()).BeginningOfHour())
}

// mergeRatesAfter blends new and existing rates, keeping existing rates after timestamp
func mergeRatesAfter(data *util.Monitor[api.Rates], new api.Rates, now time.Time) {
	new.Sort()

	var newStart time.Time
	if len(new) > 0 {
		newStart = new[0].Start
	}

	data.SetFunc(func(old api.Rates) api.Rates {
		var between api.Rates
		for _, r := range old {
			if (r.Start.Before(newStart) && !r.Start.Before(now) || newStart.IsZero()) && r.End.After(now) {
				between = append(between, r)
			}
		}

		return append(between, new...)
	})
}

// beginningOfDay returns the beginning of the current day
func beginningOfDay() time.Time {
	return now.With(time.Now()).BeginningOfDay()
}

// hasValidSolarCoverage checks if solar forecast rates cover at least until end of day
func hasValidSolarCoverage(rates api.Rates, log *util.Logger) bool {
	// Empty data is invalid for solar forecasts
	if len(rates) == 0 {
		log.DEBUG.Printf("cached solar forecast invalid: no data")
		return false
	}

	// Find the latest end time in the rates
	var latestEnd time.Time
	for _, r := range rates {
		if r.End.After(latestEnd) {
			latestEnd = r.End
		}
	}

	// Check if rates extend to at least end of today
	endOfToday := now.With(time.Now()).EndOfDay()
	if latestEnd.Before(endOfToday) {
		log.DEBUG.Printf("cached solar forecast insufficient: covers until %v, need until %v",
			latestEnd.Format("15:04"), endOfToday.Format("15:04"))
		return false
	}

	log.DEBUG.Printf("cached solar forecast valid: covers until %v", latestEnd.Format("15:04"))
	return true
}

// loadSolarCacheWithDelay handles cache loading, validation, merging and delay for solar forecasts
// Cache contains already-processed data (post-transformation)
// Returns true if cache was used and delay applied, false if fresh fetch is needed
func loadSolarCacheWithDelay(
	cache *SolarCacheManager,
	data *util.Monitor[api.Rates],
	log *util.Logger,
	interval time.Duration,
	done chan error,
	once *sync.Once,
) bool {
	if cache == nil {
		return false
	}

	// Try to load from cache
	cached, cacheTime, ok := cache.GetWithTimestamp(interval)
	if !ok {
		return false
	}

	log.DEBUG.Printf("loaded %d rates from cache", len(cached))

	// Validate cached data has sufficient coverage
	if !hasValidSolarCoverage(cached, log) {
		return false
	}

	// Cache is valid, use it (data is already processed)
	mergeRatesAfter(data, cached, beginningOfDay())
	once.Do(func() { close(done) })

	// Calculate delay until next fetch based on cache age
	cacheAge := time.Since(cacheTime)
	if cacheAge < interval {
		initialDelay := interval - cacheAge
		log.DEBUG.Printf("delaying initial fetch by %v (cache age: %v)", initialDelay, cacheAge)
		time.Sleep(initialDelay)
	}

	return true
}
