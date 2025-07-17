package tariff

import (
	"context"
	"errors"
	"fmt"
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
	scheduleOnce sync.Once
	lastHash     atomic.Uint64
}

// NewTariffProxy creates a proxy that controls tariff instantiation and caching
func NewTariffProxy(provider string, config interface{}) (api.Tariff, error) {
	// Convert config to map[string]interface{} - required for all tariff operations
	configMap, ok := config.(map[string]interface{})
	if !ok {
		// Config must be a map - return error for setup to handle
		return nil, fmt.Errorf("invalid config type: expected map[string]interface{}, got %T", config)
	}

	// Determine the actual provider name for logging
	actualProvider := provider
	if provider == "template" {
		if template, ok := configMap["template"].(string); ok {
			actualProvider = template
		}
	}

	cache := NewSolarCacheManager(actualProvider, configMap)
	// Create logger with actual provider and config hash suffix
	loggerName := fmt.Sprintf("tariff-proxy-%s-%s", actualProvider, cache.ConfigHash[:8])

	proxy := &CachingTariffProxy{
		cache:    cache,
		provider: provider,
		config:   configMap,
		interval: extractInterval(configMap),
		log:      util.NewLogger(loggerName),
	}

	if err := proxy.init(); err != nil {
		return nil, err
	}

	return proxy, nil
}

// init initializes the proxy, creating tariff immediately if no valid cache exists
func (p *CachingTariffProxy) init() error {
	// If we already have a tariff, nothing to do
	if p.Tariff != nil {
		return nil
	}

	// Check if we have valid cached data for potential solar tariffs
	if cached, ok := p.cache.Get(24 * time.Hour); ok {
		if hasValidSolarCoverage(cached, time.Now()) {
			p.log.DEBUG.Printf("found valid cache with %d rates, delaying tariff creation", len(cached))
			// Schedule delayed tariff creation
			p.scheduleDelayedCreation()
			return nil
		}
	}

	// No valid cache - create tariff immediately to determine type
	p.log.DEBUG.Printf("no valid cache found, creating tariff immediately")
	if err := p.ensureTariff(); err != nil {
		return err
	}

	return nil
}

// ensureTariff creates the underlying tariff if not already created
func (p *CachingTariffProxy) ensureTariff() error {
	var createErr error
	p.createOnce.Do(func() {
		// Only create if we don't already have a tariff
		if p.Tariff == nil {
			ctx := util.WithLogger(context.Background(), p.log)
			tariff, err := NewFromConfig(ctx, p.provider, p.config)
			if err != nil {
				if ce := new(util.ConfigError); errors.As(err, &ce) {
					createErr = err
					return
				}
				// wrap non-config tariff errors to prevent fatals
				p.log.ERROR.Printf("creating tariff failed: %v", err)
				p.Tariff = NewWrapper(p.provider, p.config, err)
			} else {
				p.Tariff = tariff
			}
		}
	})
	return createErr
}

// Rates returns cached data until underlying tariff is created, then delegates to tariff
func (p *CachingTariffProxy) Rates() (api.Rates, error) {
	// If tariff is already created, delegate to it
	if p.Tariff != nil {
		rates, err := p.Tariff.Rates()
		if err == nil && p.Tariff.Type() == api.TariffTypeSolar {
			// Only cache solar tariff data
			if p.hasDataChanged(rates) {
				if saveErr := p.cache.Set(rates, p.Tariff.Type()); saveErr != nil {
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
			p.log.TRACE.Printf("serving %d rates from cache", len(cached))
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

// Type returns the tariff type
func (p *CachingTariffProxy) Type() api.TariffType {
	// If tariff is already created, use it
	if p.Tariff != nil {
		return p.Tariff.Type()
	}

	// Try to get type from cache
	if tariffType, ok := p.cache.GetTariffType(24 * time.Hour); ok {
		return tariffType
	}

	// If no cached type and no tariff, we must create it to find out
	if err := p.ensureTariff(); err == nil && p.Tariff != nil {
		return p.Tariff.Type()
	}

	// This should never happen in normal operation
	panic("tariff proxy: cannot determine tariff type")
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

// hasDataChanged determines if rates should be cached based on content changes
func (p *CachingTariffProxy) hasDataChanged(rates api.Rates) bool {
	if len(rates) == 0 {
		return false // Don't cache empty rates
	}

	newHash := p.hashRates(rates)
	oldHash := p.lastHash.Swap(newHash)

	return newHash != oldHash
}
