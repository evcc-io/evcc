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

func TestSolarCacheManager_UniqueKeys(t *testing.T) {
	// Test that different configurations get different cache keys
	config1 := map[string]interface{}{
		"site":     "site1",
		"token":    "token1",
		"interval": "1h",
	}
	config2 := map[string]interface{}{
		"site":     "site2",
		"token":    "token2",
		"interval": "1h",
	}
	config3 := config1 // Identical config should get same key

	cache1 := NewSolarCacheManager("solcast", config1)
	cache2 := NewSolarCacheManager("solcast", config2)
	cache3 := NewSolarCacheManager("solcast", config3)

	// Different configs should have different keys
	assert.NotEqual(t, cache1.getCacheKey(), cache2.getCacheKey(),
		"Different configurations should have different cache keys")

	// Identical configs should have same key
	assert.Equal(t, cache1.getCacheKey(), cache3.getCacheKey(),
		"Identical configurations should have same cache keys")
}

func TestSolarCacheManager_MultipleProviders(t *testing.T) {
	// Test that different providers with same config get different keys
	config := map[string]interface{}{
		"lat": 52.52,
		"lon": 13.405,
		"kwp": 5,
	}

	solcastCache := NewSolarCacheManager("solcast", config)
	customCache := NewSolarCacheManager("custom", config)
	templateCache := NewSolarCacheManager("template", config)

	keys := []string{
		solcastCache.getCacheKey(),
		customCache.getCacheKey(),
		templateCache.getCacheKey(),
	}

	// All keys should be different
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			assert.NotEqual(t, keys[i], keys[j],
				"Different providers should have different cache keys")
		}
	}
}

func TestSolarCacheManager_IdenticalConfigs(t *testing.T) {
	// Test that completely identical forecast configs share cache (as intended)
	config := map[string]interface{}{
		"template": "forecast-solar",
		"lat":      52.52,
		"lon":      13.405,
		"kwp":      5,
		"interval": "1h",
	}

	// Two identical template configurations
	cache1 := NewSolarCacheManager("template", config)
	cache2 := NewSolarCacheManager("template", config)

	assert.Equal(t, cache1.getCacheKey(), cache2.getCacheKey(),
		"Identical configurations should share cache key")
	assert.Equal(t, cache1.configHash, cache2.configHash,
		"Identical configurations should have same config hash")
}

func TestSolarCacheManager_ConfigHashValidation(t *testing.T) {
	// Test cache validation logic without database dependency
	oldCache := &SolarForecastCache{
		ConfigHash: "oldConfigHash",
		Version:    util.Version,
		Timestamp:  time.Now().Add(-30 * time.Minute),
		Rates:      make(api.Rates, 5),
	}

	config := map[string]interface{}{"site": "test"}
	cache := NewSolarCacheManager("test", config)

	// Should not validate due to config hash mismatch
	isValid := cache.IsValid(oldCache, 1*time.Hour)
	assert.False(t, isValid, "Should not validate cache with mismatched config hash")
}

func TestSolarCacheManager_VersionValidation(t *testing.T) {
	// Test version validation logic
	oldCache := &SolarForecastCache{
		ConfigHash: "testHash",
		Version:    "0.200.0", // Old version
		Timestamp:  time.Now().Add(-30 * time.Minute),
		Rates:      make(api.Rates, 24),
	}

	config := map[string]interface{}{"site": "test"}
	cache := NewSolarCacheManager("test", config)
	cache.configHash = "testHash" // Match the config hash

	// Should not validate due to version mismatch
	isValid := cache.IsValid(oldCache, 1*time.Hour)
	assert.False(t, isValid, "Should not validate cache with mismatched version")
}

func TestSolarCacheManager_AgeValidation(t *testing.T) {
	// Test age validation logic
	oldCache := &SolarForecastCache{
		ConfigHash: "testHash",
		Version:    util.Version,
		Timestamp:  time.Now().Add(-2 * time.Hour), // Too old for 1 hour max age
		Rates:      make(api.Rates, 24),
	}

	config := map[string]interface{}{"site": "test"}
	cache := NewSolarCacheManager("test", config)
	cache.configHash = "testHash"

	// Should not validate due to age
	isValid := cache.IsValid(oldCache, 1*time.Hour)
	assert.False(t, isValid, "Should not validate cache when too old")
}

func TestSolarCacheManager_ValidCache(t *testing.T) {
	// Test valid cache validation logic
	testRates := api.Rates{
		{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 1000},
		{Start: time.Now().Add(time.Hour), End: time.Now().Add(2 * time.Hour), Value: 2000},
	}

	validCache := &SolarForecastCache{
		ConfigHash: "testHash",
		Version:    util.Version,
		Timestamp:  time.Now().Add(-30 * time.Minute), // Recent enough
		Rates:      testRates,
	}

	config := map[string]interface{}{"site": "test"}
	cache := NewSolarCacheManager("test", config)
	cache.configHash = "testHash"

	// Should validate
	isValid := cache.IsValid(validCache, 1*time.Hour)
	assert.True(t, isValid, "Should validate cache with matching config, version, and recent timestamp")
}

func TestSolarCacheManager_EdgeCases(t *testing.T) {
	config := map[string]interface{}{"site": "test"}
	cache := NewSolarCacheManager("test", config)

	// Test nil cache validation
	isValid := cache.IsValid(nil, 1*time.Hour)
	assert.False(t, isValid, "Should not validate nil cache")

	// Test zero rates
	emptyCache := &SolarForecastCache{
		ConfigHash: cache.configHash,
		Version:    util.Version,
		Timestamp:  time.Now(),
		Rates:      api.Rates{}, // Empty rates
	}

	isValid = cache.IsValid(emptyCache, 1*time.Hour)
	assert.True(t, isValid, "Should validate cache with empty rates")
}

func TestSolarCacheManager_ConfigHashGeneration(t *testing.T) {
	// Test that complex configs generate consistent hashes
	complexConfig := map[string]interface{}{
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": []string{"a", "b", "c"},
		},
		"simple": "value",
		"number": 123.45,
	}

	cache1 := NewSolarCacheManager("test", complexConfig)
	cache2 := NewSolarCacheManager("test", complexConfig)

	assert.Equal(t, cache1.configHash, cache2.configHash,
		"Complex configs should generate same hash")
	assert.Equal(t, cache1.getCacheKey(), cache2.getCacheKey(),
		"Same configs should have same cache key")
}

func TestSolarCacheManager_RateDataIntegrity(t *testing.T) {
	// Test that rates stored in cache exactly match what's retrieved
	originalRates := api.Rates{
		{Start: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Value: 1500.75},
		{Start: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Value: 2200.50},
		{Start: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), Value: 0.001},
	}

	config := map[string]interface{}{"site": "test"}
	cache := NewSolarCacheManager("test", config)

	// Store rates in cache
	err := cache.Set(originalRates)
	assert.NoError(t, err, "Should be able to store rates in cache")

	// Retrieve rates from cache
	retrievedRates, found := cache.Get(1 * time.Hour)
	assert.True(t, found, "Should find rates in cache")
	assert.NotNil(t, retrievedRates, "Retrieved rates should not be nil")

	// Verify exact match
	assert.Equal(t, len(originalRates), len(retrievedRates), "Rate count should match")

	for i, originalRate := range originalRates {
		assert.Equal(t, originalRate.Start, retrievedRates[i].Start, "Start time should match for rate %d", i)
		assert.Equal(t, originalRate.End, retrievedRates[i].End, "End time should match for rate %d", i)
		assert.Equal(t, originalRate.Value, retrievedRates[i].Value, "Value should match for rate %d", i)
	}
}

func TestSolarCacheManager_FloatingPointPrecision(t *testing.T) {
	// Test that floating-point precision is preserved through JSON serialization
	precisionTestRates := api.Rates{
		{Start: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Value: 1234.56789}, // Many decimal places
		{Start: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Value: 0.000001},   // Very small value
		{Start: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), Value: 999999.99},  // Large value
		{Start: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC), Value: 0.0},        // Zero value
		{Start: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC), Value: -1500.25},   // Negative value
	}

	config := map[string]interface{}{"site": "precision_test"}
	cache := NewSolarCacheManager("test", config)

	// Store rates
	err := cache.Set(precisionTestRates)
	assert.NoError(t, err, "Should be able to store precision test rates")

	// Retrieve rates
	retrievedRates, found := cache.Get(1 * time.Hour)
	assert.True(t, found, "Should find precision test rates in cache")
	assert.Equal(t, len(precisionTestRates), len(retrievedRates), "Rate count should match")

	// Verify precise floating-point values
	for i, originalRate := range precisionTestRates {
		assert.Equal(t, originalRate.Value, retrievedRates[i].Value,
			"Floating-point value should be preserved exactly for rate %d (expected: %v, got: %v)",
			i, originalRate.Value, retrievedRates[i].Value)
	}
}

func TestSolarCacheManager_TimestampPrecision(t *testing.T) {
	// Test that timestamp precision is preserved including nanoseconds
	now := time.Now()
	timestampTestRates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 1000.0},
		{Start: now.Add(time.Millisecond * 123), End: now.Add(time.Hour + time.Millisecond*456), Value: 2000.0},
		{Start: now.Add(time.Microsecond * 789), End: now.Add(time.Hour + time.Microsecond*987), Value: 3000.0},
		{Start: now.Add(time.Nanosecond * 654321), End: now.Add(time.Hour + time.Nanosecond*123456), Value: 4000.0},
	}

	config := map[string]interface{}{"site": "timestamp_test"}
	cache := NewSolarCacheManager("test", config)

	// Store rates
	err := cache.Set(timestampTestRates)
	assert.NoError(t, err, "Should be able to store timestamp test rates")

	// Retrieve rates
	retrievedRates, found := cache.Get(1 * time.Hour)
	assert.True(t, found, "Should find timestamp test rates in cache")
	assert.Equal(t, len(timestampTestRates), len(retrievedRates), "Rate count should match")

	// Verify timestamp precision
	for i, originalRate := range timestampTestRates {
		assert.True(t, originalRate.Start.Equal(retrievedRates[i].Start),
			"Start timestamp should be preserved exactly for rate %d (expected: %v, got: %v)",
			i, originalRate.Start, retrievedRates[i].Start)
		assert.True(t, originalRate.End.Equal(retrievedRates[i].End),
			"End timestamp should be preserved exactly for rate %d (expected: %v, got: %v)",
			i, originalRate.End, retrievedRates[i].End)
	}
}

func TestSolarCacheManager_LargeDatasetHandling(t *testing.T) {
	// Test that large datasets can be stored and retrieved correctly
	const numRates = 1000
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Generate large dataset with varying values
	largeDataset := make(api.Rates, numRates)
	for i := 0; i < numRates; i++ {
		largeDataset[i] = api.Rate{
			Start: baseTime.Add(time.Duration(i) * time.Hour),
			End:   baseTime.Add(time.Duration(i+1) * time.Hour),
			Value: float64(i)*10.5 + 100.25, // Varied floating-point values
		}
	}

	config := map[string]interface{}{"site": "large_dataset_test"}
	cache := NewSolarCacheManager("test", config)

	// Store large dataset
	err := cache.Set(largeDataset)
	assert.NoError(t, err, "Should be able to store large dataset")

	// Retrieve large dataset
	retrievedRates, found := cache.Get(1 * time.Hour)
	assert.True(t, found, "Should find large dataset in cache")
	assert.Equal(t, len(largeDataset), len(retrievedRates), "Large dataset count should match")

	// Verify all entries in large dataset
	for i, originalRate := range largeDataset {
		assert.Equal(t, originalRate.Start, retrievedRates[i].Start, "Start time should match for large dataset rate %d", i)
		assert.Equal(t, originalRate.End, retrievedRates[i].End, "End time should match for large dataset rate %d", i)
		assert.Equal(t, originalRate.Value, retrievedRates[i].Value, "Value should match for large dataset rate %d", i)
	}
}

func TestSolarCacheManager_EdgeCasesDataIntegrity(t *testing.T) {
	// Test edge cases for data integrity
	config := map[string]interface{}{"site": "edge_cases_test"}
	cache := NewSolarCacheManager("test", config)

	// Test 1: Empty rates slice
	emptyRates := api.Rates{}
	err := cache.Set(emptyRates)
	assert.NoError(t, err, "Should be able to store empty rates")

	retrievedEmpty, found := cache.Get(1 * time.Hour)
	assert.True(t, found, "Should find empty rates in cache")
	assert.Equal(t, len(emptyRates), len(retrievedEmpty), "Empty rates count should match")
	assert.Equal(t, emptyRates, retrievedEmpty, "Empty rates should match exactly")

	// Test 2: Single rate with zero value
	singleZeroRate := api.Rates{
		{Start: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Value: 0.0},
	}
	err = cache.Set(singleZeroRate)
	assert.NoError(t, err, "Should be able to store single zero rate")

	retrievedSingle, found := cache.Get(1 * time.Hour)
	assert.True(t, found, "Should find single zero rate in cache")
	assert.Equal(t, len(singleZeroRate), len(retrievedSingle), "Single rate count should match")
	assert.Equal(t, singleZeroRate[0].Value, retrievedSingle[0].Value, "Zero value should be preserved")

	// Test 3: Rates with extreme values
	extremeRates := api.Rates{
		{Start: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), Value: -999999.999},
		{Start: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Value: 999999.999},
		{Start: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), Value: 1e-10},
		{Start: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC), Value: 1e10},
	}
	err = cache.Set(extremeRates)
	assert.NoError(t, err, "Should be able to store extreme value rates")

	retrievedExtreme, found := cache.Get(1 * time.Hour)
	assert.True(t, found, "Should find extreme value rates in cache")
	assert.Equal(t, len(extremeRates), len(retrievedExtreme), "Extreme rates count should match")

	for i, originalRate := range extremeRates {
		assert.Equal(t, originalRate.Value, retrievedExtreme[i].Value,
			"Extreme value should be preserved for rate %d (expected: %v, got: %v)",
			i, originalRate.Value, retrievedExtreme[i].Value)
	}
}

func TestSolarCacheManager_DatabaseRoundtrip(t *testing.T) {
	// Setup in-memory database for testing
	cleanup := setupTestDB(t)
	defer cleanup()

	// Test data with various characteristics
	originalRates := api.Rates{
		{Start: time.Date(2024, 1, 1, 10, 0, 0, 123456789, time.UTC), End: time.Date(2024, 1, 1, 11, 0, 0, 987654321, time.UTC), Value: 1234.56789},
		{Start: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Value: 0.000001},
		{Start: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), End: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), Value: -1500.75},
	}

	config := map[string]interface{}{
		"site": "database_test",
		"key":  "unique_value_123",
	}

	// Create cache manager
	cache := NewSolarCacheManager("test", config)

	// Store rates in cache (should persist to database)
	err := cache.Set(originalRates)
	assert.NoError(t, err, "Should be able to store rates in database")

	// Verify data was stored - create new cache manager instance
	cache2 := NewSolarCacheManager("test", config)

	// Retrieve from database
	retrievedRates, found := cache2.Get(1 * time.Hour)
	assert.True(t, found, "Should find rates in database")
	assert.Equal(t, len(originalRates), len(retrievedRates), "Database roundtrip should preserve rate count")

	// Verify complete data integrity through database roundtrip
	for i, originalRate := range originalRates {
		assert.True(t, originalRate.Start.Equal(retrievedRates[i].Start),
			"Database roundtrip should preserve start time for rate %d (expected: %v, got: %v)",
			i, originalRate.Start, retrievedRates[i].Start)
		assert.True(t, originalRate.End.Equal(retrievedRates[i].End),
			"Database roundtrip should preserve end time for rate %d (expected: %v, got: %v)",
			i, originalRate.End, retrievedRates[i].End)
		assert.Equal(t, originalRate.Value, retrievedRates[i].Value,
			"Database roundtrip should preserve value for rate %d (expected: %v, got: %v)",
			i, originalRate.Value, retrievedRates[i].Value)
	}

	// Test that different configs get different cache keys
	differentConfig := map[string]interface{}{
		"site": "database_test",
		"key":  "different_value_456",
	}
	cache3 := NewSolarCacheManager("test", differentConfig)

	// Should not find cached data for different config
	_, found = cache3.Get(1 * time.Hour)
	assert.False(t, found, "Different config should not find cached data")

	// Test cache expiration through database
	// Note: The Set() method always uses time.Now() for timestamp, so we can't directly test
	// expiration by storing old data. Instead, we test with a very short maxAge.
	testRates := api.Rates{
		{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 999.0},
	}

	cache4 := NewSolarCacheManager("expiration_test", config)
	err = cache4.Set(testRates)
	assert.NoError(t, err, "Should be able to store test rates")

	// Sleep briefly to ensure cache timestamp is older than maxAge
	time.Sleep(10 * time.Millisecond)

	// Should not find data when maxAge is very small
	_, found = cache4.Get(5 * time.Millisecond) // Max age less than actual age
	assert.False(t, found, "Should not find cached data when maxAge is exceeded")

	// Dump database contents to see what's actually stored
	t.Log("=== Database Contents ===")
	allSettings := settings.All()
	t.Logf("Total settings in database: %d", len(allSettings))

	for _, setting := range allSettings {
		if strings.HasPrefix(setting.Key, "solar_forecast_cache_") {
			t.Logf("Cache Key: %s", setting.Key)
			t.Logf("Raw Value: %s", setting.Value)

			// Try to parse the JSON to see the structure
			var cached SolarForecastCache
			if err := json.Unmarshal([]byte(setting.Value), &cached); err == nil {
				t.Logf("  ConfigHash: %s", cached.ConfigHash)
				t.Logf("  Version: %s", cached.Version)
				t.Logf("  Timestamp: %s", cached.Timestamp.Format(time.RFC3339Nano))
				t.Logf("  Number of Rates: %d", len(cached.Rates))
				if len(cached.Rates) > 0 {
					for i, rate := range cached.Rates {
						if i < 3 { // Only show first 3 rates to avoid spam
							t.Logf("    Rate[%d]: Start=%s, End=%s, Value=%f",
								i, rate.Start.Format(time.RFC3339Nano), rate.End.Format(time.RFC3339Nano), rate.Value)
						}
					}
					if len(cached.Rates) > 3 {
						t.Logf("    ... and %d more rates", len(cached.Rates)-3)
					}
				}
			} else {
				t.Logf("  JSON Parse Error: %v", err)
			}
			t.Log("---")
		}
	}
}

func setupTestDB(t *testing.T) func() {
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
