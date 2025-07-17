package tariff

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
)

// SolarForecastCache represents cached solar forecast data
type SolarForecastCache struct {
	ConfigHash string         `json:"configHash"`
	Version    string         `json:"version"`
	Timestamp  time.Time      `json:"timestamp"`
	Rates      api.Rates      `json:"rates"`
	TariffType api.TariffType `json:"tariffType"`
}

// SolarCacheManager manages persistent caching for solar forecast APIs
type SolarCacheManager struct {
	log        *util.Logger
	configHash string
	version    string
	keyPrefix  string
}

// NewSolarCacheManager creates a new solar forecast cache manager
func NewSolarCacheManager(provider string, config interface{}) *SolarCacheManager {
	log := util.NewLogger("solar-cache")

	// Generate config hash for cache validation and instance identification
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.DEBUG.Printf("failed to marshal config for hash: %v", err)
		configBytes = []byte{}
	}
	configHash := fmt.Sprintf("%x", md5.Sum(configBytes))

	// Use config hash in key to ensure unique cache per provider instance
	keyPrefix := fmt.Sprintf("solar_forecast_cache_%s_%s", provider, configHash[:8])

	return &SolarCacheManager{
		log:        log,
		configHash: configHash,
		version:    util.Version,
		keyPrefix:  keyPrefix,
	}
}

// getCacheKey generates a cache key for the provider
func (c *SolarCacheManager) getCacheKey() string {
	return c.keyPrefix
}

// IsValid checks if cached data is valid based on config, version, and age
func (c *SolarCacheManager) IsValid(cached *SolarForecastCache, maxAge time.Duration) bool {
	if cached == nil {
		return false
	}

	// Check config hash
	if cached.ConfigHash != c.configHash {
		c.log.DEBUG.Printf("cache invalid: config hash mismatch (cached: %s, current: %s)",
			cached.ConfigHash[:8], c.configHash[:8])
		return false
	}

	// Check version
	if cached.Version != c.version {
		c.log.DEBUG.Printf("cache invalid: version mismatch (cached: %s, current: %s)",
			cached.Version, c.version)
		return false
	}

	// Check age
	if time.Since(cached.Timestamp) > maxAge {
		c.log.DEBUG.Printf("cache invalid: too old (age: %v, max: %v)",
			time.Since(cached.Timestamp), maxAge)
		return false
	}

	c.log.DEBUG.Printf("cache valid: age %v, %d rates",
		time.Since(cached.Timestamp), len(cached.Rates))
	return true
}

// Get retrieves cached solar forecast data
func (c *SolarCacheManager) Get(maxAge time.Duration) (api.Rates, bool) {
	var cached SolarForecastCache

	if err := settings.Json(c.getCacheKey(), &cached); err != nil {
		if err != settings.ErrNotFound {
			c.log.DEBUG.Printf("failed to load cache: %v", err)
		}
		return nil, false
	}

	if !c.IsValid(&cached, maxAge) {
		return nil, false
	}

	c.log.DEBUG.Printf("cache hit: returning %d rates", len(cached.Rates))
	return cached.Rates, true
}

// GetWithTimestamp retrieves cached solar forecast data with its timestamp
func (c *SolarCacheManager) GetWithTimestamp(maxAge time.Duration) (api.Rates, time.Time, bool) {
	var cached SolarForecastCache

	if err := settings.Json(c.getCacheKey(), &cached); err != nil {
		if err != settings.ErrNotFound {
			c.log.DEBUG.Printf("failed to load cache: %v", err)
		}
		return nil, time.Time{}, false
	}

	if !c.IsValid(&cached, maxAge) {
		return nil, time.Time{}, false
	}

	c.log.DEBUG.Printf("cache hit: returning %d rates from %v ago", len(cached.Rates), time.Since(cached.Timestamp))
	return cached.Rates, cached.Timestamp, true
}

// GetTariffType retrieves the cached tariff type
func (c *SolarCacheManager) GetTariffType(maxAge time.Duration) (api.TariffType, bool) {
	var cached SolarForecastCache

	if err := settings.Json(c.getCacheKey(), &cached); err != nil {
		if err != settings.ErrNotFound {
			c.log.DEBUG.Printf("failed to load cache: %v", err)
		}
		return 0, false
	}

	if !c.IsValid(&cached, maxAge) {
		return 0, false
	}

	return cached.TariffType, true
}

// Set stores solar forecast data in cache
func (c *SolarCacheManager) Set(rates api.Rates, tariffType api.TariffType) error {
	cached := SolarForecastCache{
		ConfigHash: c.configHash,
		Version:    c.version,
		Timestamp:  time.Now(),
		Rates:      rates,
		TariffType: tariffType,
	}

	if err := settings.SetJson(c.getCacheKey(), cached); err != nil {
		c.log.ERROR.Printf("failed to save cache: %v", err)
		return err
	}

	c.log.DEBUG.Printf("cache updated: stored %d rates", len(rates))
	return nil
}

// Clear removes cached data
func (c *SolarCacheManager) Clear() error {
	if err := settings.Delete(c.getCacheKey()); err != nil {
		c.log.ERROR.Printf("failed to clear cache: %v", err)
		return err
	}

	c.log.DEBUG.Println("cache cleared")
	return nil
}

// PruneOldCaches removes old solar forecast cache entries
func PruneOldCaches(maxAge time.Duration) error {
	log := util.NewLogger("solar-cache")

	allSettings := settings.All()
	var keysToDelete []string

	// First pass: identify keys to delete
	for _, setting := range allSettings {
		// Only process solar forecast cache keys
		if !strings.HasPrefix(setting.Key, "solar_forecast_cache_") {
			continue
		}

		var cached SolarForecastCache
		if err := json.Unmarshal([]byte(setting.Value), &cached); err != nil {
			// Invalid cache entry, mark for deletion
			log.DEBUG.Printf("marked invalid cache for deletion: %s", setting.Key)
			keysToDelete = append(keysToDelete, setting.Key)
			continue
		}

		// Check if cache is too old
		if time.Since(cached.Timestamp) > maxAge {
			log.DEBUG.Printf("marked old cache for deletion: %s (age: %v)", setting.Key, time.Since(cached.Timestamp))
			keysToDelete = append(keysToDelete, setting.Key)
			continue
		}

		// Check if version is outdated
		if cached.Version != util.Version {
			log.DEBUG.Printf("marked outdated cache for deletion: %s (version: %s)", setting.Key, cached.Version)
			keysToDelete = append(keysToDelete, setting.Key)
			continue
		}
	}

	// Second pass: delete the marked keys
	var deleted int
	for _, key := range keysToDelete {
		if err := settings.Delete(key); err != nil {
			log.DEBUG.Printf("failed to delete cache %s: %v", key, err)
		} else {
			deleted++
		}
	}

	if deleted > 0 {
		log.INFO.Printf("pruned %d old solar forecast cache entries", deleted)
	}

	return nil
}
