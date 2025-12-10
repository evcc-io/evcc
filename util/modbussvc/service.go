package modbussvc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

var log = util.NewLogger("modbus")

// Simple cache for service responses
type cacheEntry struct {
	value     any
	timestamp time.Time
}

var (
	cache      = make(map[string]cacheEntry)
	cacheMutex sync.RWMutex
	cacheTTL   = 1 * time.Minute // Cache for 1 minute
)

// Query combines modbus settings, register config, and additional parameters
type Query struct {
	modbus.Settings `mapstructure:",squash"`
	modbus.Register `mapstructure:",squash"`
	Scale           float64 `mapstructure:"scale"` // scaling factor
	Cast            string  `mapstructure:"cast"`  // type cast (int, float, string)
}

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /params", getParams)

	service.Register("modbus", mux)
}

// getParams reads a parameter value from a device based on URL parameters
// Returns single value as array (for UI compatibility)
func getParams(w http.ResponseWriter, req *http.Request) {
	// Convert URL query parameters to map for decoding
	cc := make(map[string]any)
	for k := range req.URL.Query() {
		cc[k] = req.URL.Query().Get(k)
	}

	// Decode query parameters into Query struct using mapstructure
	query := Query{Scale: 1.0}
	if err := util.DecodeOther(cc, &query); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// Create cache key from URI and register address
	cacheKey := fmt.Sprintf("%s:%d", query.URI, query.Address)

	// Check cache first
	cacheMutex.RLock()
	if entry, ok := cache[cacheKey]; ok && time.Since(entry.timestamp) < cacheTTL {
		cacheMutex.RUnlock()
		log.TRACE.Printf("Cache hit for %s", cacheKey)
		finalValue := applyCast(entry.value, query.Cast)
		valueStr := fmt.Sprintf("%v", finalValue)
		jsonWrite(w, []string{valueStr})
		return
	}
	cacheMutex.RUnlock()

	// Read value from modbus using plugin
	// Use background context so connection isn't tied to HTTP request lifecycle
	value, err := readRegisterValue(context.TODO(), query.Settings, query.Register, query.Scale)
	if err != nil {
		log.DEBUG.Printf("Failed to read register %d: %v", query.Address, err)
		jsonWrite(w, []string{}) // Return empty array on error
		return
	}

	// Store in cache
	cacheMutex.Lock()
	cache[cacheKey] = cacheEntry{
		value:     value,
		timestamp: time.Now(),
	}
	cacheMutex.Unlock()

	// Apply cast if specified
	finalValue := applyCast(value, query.Cast)

	// Return value as string array (for UI compatibility)
	valueStr := fmt.Sprintf("%v", finalValue)
	jsonWrite(w, []string{valueStr})
}

// readRegisterValue reads a modbus register value by reusing the modbus plugin
func readRegisterValue(ctx context.Context, settings modbus.Settings, reg modbus.Register, scale float64) (any, error) {
	// Build config for plugin
	cfg := map[string]any{
		"uri":      settings.URI,
		"id":       settings.ID,
		"register": reg,
		"scale":    scale,
	}

	// Add optional settings
	if settings.Device != "" {
		cfg["device"] = settings.Device
	}
	if settings.Comset != "" {
		cfg["comset"] = settings.Comset
	}
	if settings.Baudrate != 0 {
		cfg["baudrate"] = settings.Baudrate
	}

	// Create plugin instance (reuses connection pool automatically)
	p, err := plugin.NewModbusFromConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create modbus plugin: %w", err)
	}

	// Try different getters based on what the plugin supports
	if fg, ok := p.(plugin.FloatGetter); ok {
		getter, err := fg.FloatGetter()
		if err != nil {
			return nil, err
		}
		return getter()
	}

	if ig, ok := p.(plugin.IntGetter); ok {
		getter, err := ig.IntGetter()
		if err != nil {
			return nil, err
		}
		return getter()
	}

	if bg, ok := p.(plugin.BoolGetter); ok {
		getter, err := bg.BoolGetter()
		if err != nil {
			return nil, err
		}
		return getter()
	}

	if sg, ok := p.(plugin.StringGetter); ok {
		getter, err := sg.StringGetter()
		if err != nil {
			return nil, err
		}
		return getter()
	}

	return nil, fmt.Errorf("plugin does not implement any supported getter interface")
}

// applyCast applies type casting to the value
func applyCast(value any, cast string) any {
	if cast == "" {
		return value
	}

	switch cast {
	case "int":
		if v, ok := value.(float64); ok {
			return int64(v)
		}
	case "float":
		if v, ok := value.(int64); ok {
			return float64(v)
		}
	case "string":
		return fmt.Sprintf("%v", value)
	}

	return value
}

// jsonWrite writes a JSON response
func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error response
func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
