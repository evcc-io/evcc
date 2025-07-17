package tariff

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCleanupTestDB(t *testing.T) func() {
	// Use in-memory SQLite database for testing
	err := db.NewInstance("sqlite", ":memory:")
	require.NoError(t, err, "Failed to create test database")

	err = settings.Init()
	require.NoError(t, err, "Failed to initialize settings")

	// Return cleanup function
	return func() {
		// Reset the database instance
		db.Instance = nil
	}
}

// TestPruneOldCaches_Cleanup_Integration tests the complete solar cache cleanup functionality
// This test demonstrates that the settings delete bug fix enables proper cache cleanup
func TestPruneOldCaches_Cleanup_Integration(t *testing.T) {
	cleanup := setupCleanupTestDB(t)
	defer cleanup()

	// Create test cache entries with different conditions that should trigger cleanup
	testEntries := []struct {
		key         string
		age         time.Duration
		version     string
		valid       bool
		shouldPrune bool
		description string
	}{
		{
			key:         "solar_forecast_cache_very_old",
			age:         48 * time.Hour, // Very old (2 days)
			version:     util.Version,
			valid:       true,
			shouldPrune: true,
			description: "very old cache entry",
		},
		{
			key:         "solar_forecast_cache_old",
			age:         25 * time.Hour, // Old (1+ day)
			version:     util.Version,
			valid:       true,
			shouldPrune: true,
			description: "old cache entry",
		},
		{
			key:         "solar_forecast_cache_recent",
			age:         2 * time.Hour, // Recent
			version:     util.Version,
			valid:       true,
			shouldPrune: false,
			description: "recent cache entry",
		},
		{
			key:         "solar_forecast_cache_wrong_version",
			age:         6 * time.Hour, // Recent but wrong version
			version:     "0.190.0",
			valid:       true,
			shouldPrune: true,
			description: "wrong version cache entry",
		},
		{
			key:         "solar_forecast_cache_invalid_json",
			age:         1 * time.Hour, // Recent but invalid
			version:     util.Version,
			valid:       false,
			shouldPrune: true,
			description: "invalid JSON cache entry",
		},
		{
			key:         "regular_setting",
			age:         0, // Not a cache entry
			version:     "",
			valid:       false,
			shouldPrune: false,
			description: "non-cache setting",
		},
	}

	// Create the test entries
	for _, entry := range testEntries {
		if entry.key == "regular_setting" {
			// Create a regular setting (not a solar cache)
			settings.SetString(entry.key, "some regular configuration value")
		} else if !entry.valid {
			// Create invalid JSON
			settings.SetString(entry.key, "invalid json data")
		} else {
			// Create valid cache entry
			cache := SolarForecastCache{
				ConfigHash: "test_config_hash",
				Version:    entry.version,
				Timestamp:  time.Now().Add(-entry.age),
				Rates: api.Rates{
					{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 1000},
					{Start: time.Now().Add(time.Hour), End: time.Now().Add(2 * time.Hour), Value: 1500},
				},
				TariffType: api.TariffTypeSolar,
			}
			cacheBytes, err := json.Marshal(cache)
			require.NoError(t, err, "Failed to marshal cache for %s", entry.description)
			settings.SetString(entry.key, string(cacheBytes))
		}
	}

	// Verify all entries exist before cleanup
	for _, entry := range testEntries {
		_, err := settings.String(entry.key)
		assert.NoError(t, err, "Entry %s should exist before cleanup", entry.key)
	}

	// Count total entries before cleanup
	allBefore := settings.All()
	totalBefore := len(allBefore)
	solarCachesBefore := 0
	for _, setting := range allBefore {
		if strings.HasPrefix(setting.Key, "solar_forecast_cache") {
			solarCachesBefore++
		}
	}
	assert.Equal(t, 5, solarCachesBefore, "Should have 5 solar cache entries before cleanup")

	// Run the cleanup with 24 hour max age
	err := PruneOldCaches(24 * time.Hour)
	require.NoError(t, err, "PruneOldCaches should not return an error")

	// Verify the cleanup results
	prunedCount := 0
	keptCount := 0

	for _, entry := range testEntries {
		_, err := settings.String(entry.key)
		exists := err == nil

		if entry.shouldPrune {
			assert.False(t, exists, "Entry %s (%s) should have been pruned", entry.key, entry.description)
			if !exists {
				prunedCount++
			}
		} else {
			assert.True(t, exists, "Entry %s (%s) should have been kept", entry.key, entry.description)
			if exists {
				keptCount++
			}
		}
	}

	// Verify expected counts
	expectedPruned := 4 // very_old, old, wrong_version, invalid_json
	expectedKept := 2   // recent, regular_setting

	assert.Equal(t, expectedPruned, prunedCount, "Should have pruned %d entries", expectedPruned)
	assert.Equal(t, expectedKept, keptCount, "Should have kept %d entries", expectedKept)

	// Verify total count in settings has decreased
	allAfter := settings.All()
	totalAfter := len(allAfter)
	assert.Equal(t, totalBefore-expectedPruned, totalAfter, "Total settings count should decrease by %d", expectedPruned)

	// Verify only 1 solar cache remains (the recent one)
	solarCachesAfter := 0
	for _, setting := range allAfter {
		if strings.HasPrefix(setting.Key, "solar_forecast_cache") {
			solarCachesAfter++
		}
	}
	assert.Equal(t, 1, solarCachesAfter, "Should have 1 solar cache entry after cleanup")
}

// TestPruneOldCaches_EdgeCases_Integration tests edge cases for cache cleanup
func TestPruneOldCaches_EdgeCases_Integration(t *testing.T) {
	cleanup := setupCleanupTestDB(t)
	defer cleanup()

	t.Run("empty_database", func(t *testing.T) {
		// Test cleanup on empty database
		err := PruneOldCaches(24 * time.Hour)
		assert.NoError(t, err, "Cleanup should work on empty database")

		all := settings.All()
		assert.Len(t, all, 0, "Should remain empty after cleanup")
	})

	t.Run("no_solar_caches", func(t *testing.T) {
		// Add only non-solar-cache settings
		settings.SetString("user_config", "value1")
		settings.SetString("app_setting", "value2")
		settings.SetString("other_cache_type", "value3")

		err := PruneOldCaches(24 * time.Hour)
		assert.NoError(t, err, "Cleanup should work with no solar caches")

		// All settings should remain
		val1, err1 := settings.String("user_config")
		assert.NoError(t, err1)
		assert.Equal(t, "value1", val1)

		val2, err2 := settings.String("app_setting")
		assert.NoError(t, err2)
		assert.Equal(t, "value2", val2)

		val3, err3 := settings.String("other_cache_type")
		assert.NoError(t, err3)
		assert.Equal(t, "value3", val3)
	})

	t.Run("zero_max_age", func(t *testing.T) {
		// Create a very recent cache entry
		cache := SolarForecastCache{
			ConfigHash: "recent_hash",
			Version:    util.Version,
			Timestamp:  time.Now().Add(-1 * time.Second), // Very recent
			Rates:      api.Rates{{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 500}},
			TariffType: api.TariffTypeSolar,
		}
		cacheBytes, _ := json.Marshal(cache)
		settings.SetString("solar_forecast_cache_recent", string(cacheBytes))

		// With zero max age, even recent entries should be pruned
		err := PruneOldCaches(0)
		assert.NoError(t, err, "Cleanup should work with zero max age")

		_, err = settings.String("solar_forecast_cache_recent")
		assert.Error(t, err, "Even recent entry should be pruned with zero max age")
	})

	t.Run("very_large_max_age", func(t *testing.T) {
		// Create an old cache entry
		cache := SolarForecastCache{
			ConfigHash: "old_hash",
			Version:    util.Version,
			Timestamp:  time.Now().Add(-100 * time.Hour), // Very old
			Rates:      api.Rates{{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 500}},
			TariffType: api.TariffTypeSolar,
		}
		cacheBytes, _ := json.Marshal(cache)
		settings.SetString("solar_forecast_cache_old", string(cacheBytes))

		// With very large max age, even old entries should be kept
		err := PruneOldCaches(365 * 24 * time.Hour) // 1 year
		assert.NoError(t, err, "Cleanup should work with large max age")

		val, err := settings.String("solar_forecast_cache_old")
		assert.NoError(t, err, "Old entry should be kept with very large max age")
		assert.NotEmpty(t, val, "Value should not be empty")
	})
}

// TestPruneOldCaches_RealWorld_Integration simulates real-world cache cleanup scenarios
func TestPruneOldCaches_RealWorld_Integration(t *testing.T) {
	cleanup := setupCleanupTestDB(t)
	defer cleanup()

	// Simulate a real scenario: multiple cache instances from different providers and configurations
	testCaches := []struct {
		provider string
		config   map[string]interface{}
		age      time.Duration
	}{
		{"solcast", map[string]interface{}{"site": "site1", "token": "token1"}, 30 * time.Hour}, // Old
		{"solcast", map[string]interface{}{"site": "site2", "token": "token2"}, 2 * time.Hour},  // Recent
		{"custom", map[string]interface{}{"lat": 52.52, "lon": 13.405}, 36 * time.Hour},         // Old
		{"custom", map[string]interface{}{"lat": 51.52, "lon": 12.405}, 1 * time.Hour},          // Recent
		{"template", map[string]interface{}{"template": "forecast-solar"}, 48 * time.Hour},      // Very old
	}

	// Create cache managers and populate with test data
	for i, tc := range testCaches {
		cache := NewSolarCacheManager(tc.provider, tc.config)

		// Create test rates
		testRates := api.Rates{
			{Start: time.Now(), End: time.Now().Add(time.Hour), Value: float64(1000 + i*100)},
		}

		// Manually create cache entry with specific timestamp
		cacheEntry := SolarForecastCache{
			ConfigHash: cache.configHash,
			Version:    util.Version,
			Timestamp:  time.Now().Add(-tc.age),
			Rates:      testRates,
			TariffType: api.TariffTypeSolar,
		}
		cacheBytes, err := json.Marshal(cacheEntry)
		require.NoError(t, err)
		settings.SetString(cache.getCacheKey(), string(cacheBytes))
	}

	// Verify all caches exist
	totalCaches := 0
	for _, tc := range testCaches {
		cache := NewSolarCacheManager(tc.provider, tc.config)
		_, err := settings.String(cache.getCacheKey())
		assert.NoError(t, err, "Cache should exist before cleanup")
		totalCaches++
	}
	assert.Equal(t, 5, totalCaches, "Should have 5 total caches")

	// Run cleanup with 24 hour max age
	err := PruneOldCaches(24 * time.Hour)
	require.NoError(t, err, "Cleanup should not error")

	// Check results - should keep recent ones (< 24h), prune old ones (> 24h)
	expectedKept := 0
	expectedPruned := 0

	for _, tc := range testCaches {
		cache := NewSolarCacheManager(tc.provider, tc.config)
		_, err := settings.String(cache.getCacheKey())
		exists := err == nil

		if tc.age > 24*time.Hour {
			assert.False(t, exists, "Cache older than 24h should be pruned (age: %v)", tc.age)
			expectedPruned++
		} else {
			assert.True(t, exists, "Cache newer than 24h should be kept (age: %v)", tc.age)
			expectedKept++
		}
	}

	assert.Equal(t, 2, expectedKept, "Should keep 2 recent caches")
	assert.Equal(t, 3, expectedPruned, "Should prune 3 old caches")
}
