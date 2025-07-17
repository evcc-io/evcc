package tariff

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// CachingTariffProxy wraps a tariff provider to add caching functionality
type CachingTariffProxy struct {
	api.Tariff   // Will be nil initially
	cache        *SolarCacheManager
	provider     string
	config       map[string]interface{}
	interval     time.Duration
	log          *util.Logger
	createOnce   sync.Once
	createErr    error
	scheduleOnce sync.Once
	lastHash     atomic.Uint64
}

// NewCachingTariffProxy creates a new caching proxy for solar tariffs
func NewCachingTariffProxy(tariff api.Tariff, provider string, config interface{}) api.Tariff {
	// For non-solar tariffs, return the original tariff unchanged
	if tariff != nil && tariff.Type() != api.TariffTypeSolar {
		return tariff
	}

	// For solar tariffs, we'll delay creation
	// First, convert config to map[string]interface{} if needed
	configMap, ok := config.(map[string]interface{})
	if !ok {
		// If we can't convert, fall back to immediate creation
		return tariff
	}

	proxy := &CachingTariffProxy{
		Tariff:   tariff, // Might be nil if we're called differently later
		cache:    NewSolarCacheManager(provider, configMap),
		provider: provider,
		config:   configMap,
		interval: extractInterval(configMap),
		log:      util.NewLogger("tariff-cache"),
	}
	
	proxy.init()
	return proxy
}


// init initializes the proxy, creating tariff immediately if no valid cache exists
func (p *CachingTariffProxy) init() {
	// If we already have a tariff, nothing to do
	if p.Tariff != nil {
		return
	}
	
	// Check if we have valid cached data for solar tariffs
	if cached, ok := p.cache.Get(24 * time.Hour); ok {
		if hasValidSolarCoverage(cached, time.Now()) {
			p.log.DEBUG.Printf("found valid cache with %d rates, delaying tariff creation", len(cached))
			return
		}
	}
	
	// No valid cache - create tariff immediately to determine type
	p.log.DEBUG.Printf("no valid cache found, creating tariff immediately")
	if err := p.ensureTariff(); err != nil {
		p.log.ERROR.Printf("failed to create tariff during init: %v", err)
		p.createErr = err
	}
}

// ensureTariff creates the underlying tariff if not already created
func (p *CachingTariffProxy) ensureTariff() error {
	p.createOnce.Do(func() {
		// Only create if we don't already have a tariff
		if p.Tariff == nil {
			ctx := util.WithLogger(context.Background(), p.log)
			p.Tariff, p.createErr = NewFromConfig(ctx, p.provider, p.config)
		}
	})
	return p.createErr
}

// Rates returns cached data until underlying tariff is created, then delegates to tariff
func (p *CachingTariffProxy) Rates() (api.Rates, error) {
	// If tariff is already created, delegate to it
	if p.Tariff != nil {
		rates, err := p.Tariff.Rates()
		if err == nil && p.Tariff.Type() == api.TariffTypeSolar {
			// Only cache solar tariff data
			if p.shouldCache(rates) {
				if saveErr := p.cache.Set(rates); saveErr != nil {
					p.log.DEBUG.Printf("failed to cache rates: %v", saveErr)
				} else {
					p.log.DEBUG.Printf("cached %d rates (data changed)", len(rates))
				}
			} else {
				p.log.TRACE.Printf("skipped caching %d rates (data unchanged)", len(rates))
			}
		}
		return rates, err
	}

	// Tariff not created yet - try cache first (only for potential solar tariffs)
	if cached, ok := p.cache.Get(24 * time.Hour); ok {
		if hasValidSolarCoverage(cached, time.Now()) {
			p.log.DEBUG.Printf("serving %d rates from cache", len(cached))

			// Schedule delayed tariff creation
			p.scheduleDelayedCreation()

			return cached, nil
		}
	}

	// No valid cache and no tariff - create tariff immediately
	if err := p.ensureTariff(); err != nil {
		return nil, err
	}

	// Now delegate to the newly created tariff
	return p.Rates()
}

// solarProviders contains known solar tariff providers
var solarProviders = map[string]bool{
	"solcast": true,
	// Add other solar providers here as they're added
}

// Type returns the tariff type
func (p *CachingTariffProxy) Type() api.TariffType {
	// If tariff is already created, use it
	if p.Tariff != nil {
		return p.Tariff.Type()
	}

	// Otherwise, infer from provider name
	// This is only used during the delay period
	if solarProviders[p.provider] {
		return api.TariffTypeSolar
	}

	// For custom tariffs, check the config
	if tariffType, ok := p.config["tariff"].(string); ok && tariffType == "solar" {
		return api.TariffTypeSolar
	}

	// Default to solar for safety (since we only wrap solar tariffs)
	return api.TariffTypeSolar
}

// scheduleDelayedCreation schedules the tariff creation based on cache age
func (p *CachingTariffProxy) scheduleDelayedCreation() {
	p.scheduleOnce.Do(func() {
		go func() {
			delay := p.calculateCreationDelay()
			if delay > 0 {
				p.log.DEBUG.Printf("delaying tariff creation by %v", delay)
				time.Sleep(delay)
			}

			// Create the tariff
			if err := p.ensureTariff(); err != nil {
				p.log.ERROR.Printf("failed to create tariff: %v", err)
			}
		}()
	})
}

// extractInterval extracts the interval from config, with defaults per provider
func extractInterval(config map[string]interface{}) time.Duration {
	if intervalStr, ok := config["interval"].(string); ok {
		if duration, err := time.ParseDuration(intervalStr); err == nil {
			return duration
		}
	}
	
	return time.Hour
}

// calculateCreationDelay calculates delay based on cache age and interval
func (p *CachingTariffProxy) calculateCreationDelay() time.Duration {
	// Get cache with timestamp to calculate age
	if _, timestamp, ok := p.cache.GetWithTimestamp(24 * time.Hour); ok {
		cacheAge := time.Since(timestamp)

		// If cache age is less than the interval, delay until interval is reached
		if cacheAge < p.interval {
			return p.interval - cacheAge
		}
	}

	// No delay needed
	return 0
}


// hashRates calculates FNV-64a hash of the rates data using unsafe for performance
func (p *CachingTariffProxy) hashRates(rates api.Rates) uint64 {
	if len(rates) == 0 {
		return 0
	}

	h := fnv.New64a()

	// Hash the entire rates slice as raw bytes
	// This is safe for runtime-only change detection within the same process
	rateSize := unsafe.Sizeof(rates[0])
	totalBytes := uintptr(len(rates)) * rateSize

	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&rates[0])), totalBytes)
	h.Write(bytes)

	return h.Sum64()
}

// shouldCache determines if rates should be cached based on content changes
func (p *CachingTariffProxy) shouldCache(rates api.Rates) bool {
	if len(rates) == 0 {
		return false // Don't cache empty rates
	}

	newHash := p.hashRates(rates)
	oldHash := p.lastHash.Swap(newHash)

	return newHash != oldHash
}
