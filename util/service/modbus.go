package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/spf13/cast"
)

// Simple cache for service responses
type cacheEntry struct {
	value     any
	timestamp time.Time
}

var (
	cache    = make(map[string]cacheEntry)
	mu       sync.RWMutex
	cacheTTL = 1 * time.Minute // Cache for 1 minute
)

// Query combines modbus settings, register config, and additional parameters
type Query struct {
	modbus.Settings `mapstructure:",squash"`
	modbus.Register `mapstructure:",squash"`
	Scale           float64 // scaling factor
	ResultType      string  // type cast (int, float, string)
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
	query := Query{
		Scale: 1.0,
	}

	if err := util.DecodeOther(cc, &query); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// Validate required parameters
	if (query.URI == "" && query.Device == "") || query.Address == 0 {
		jsonError(w, http.StatusBadRequest, fmt.Errorf("uri or device and address parameters are required"))
		return
	}

	// Create cache key from connection string and register address
	connStr := query.URI
	if connStr == "" {
		connStr = query.Device
	}
	cacheKey := fmt.Sprintf("%s:%d", connStr, query.Address)

	// Check cache first
	mu.RLock()
	if entry, ok := cache[cacheKey]; ok && time.Since(entry.timestamp) < cacheTTL {
		mu.RUnlock()
		jsonWrite(w, []string{cast.ToString(entry.value)})
		return
	}
	mu.RUnlock()

	// Read value from modbus using plugin
	// Use background context so connection isn't tied to HTTP request lifecycle
	value, err := readRegisterValue(context.TODO(), query)
	if err != nil {
		jsonWrite(w, []string{}) // Return empty array on error
		return
	}

	// Apply optional cast
	if query.ResultType != "" {
		value = applyCast(value, query.ResultType)
	}

	// Store in cache
	mu.Lock()
	cache[cacheKey] = cacheEntry{
		value:     value,
		timestamp: time.Now(),
	}
	mu.Unlock()

	jsonWrite(w, []string{cast.ToString(value)})
}

// readRegisterValue reads a modbus register value by reusing the modbus plugin
func readRegisterValue(ctx context.Context, query Query) (res any, err error) {
	// Build config map for plugin - need to flatten embedded structs manually
	cfg := map[string]any{
		"uri":      query.URI,
		"id":       query.ID,
		"register": query.Register,
		"scale":    query.Scale,
	}

	// Add optional settings
	if query.Device != "" {
		cfg["device"] = query.Device
	}
	if query.Comset != "" {
		cfg["comset"] = query.Comset
	}
	if query.Baudrate != 0 {
		cfg["baudrate"] = query.Baudrate
	}

	p, err := plugin.NewModbusFromConfig(ctx, cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to create modbus plugin: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			res = nil
			err = fmt.Errorf("read failed: %v", r)
		}
	}()

	// Choose getter based on encoding type
	encoding := strings.ToLower(query.Encoding)

	// String encodings need special handling
	if encoding == "string" || encoding == "bytes" {
		return callGetter(p.(plugin.StringGetter).StringGetter())
	}

	// For all numeric encodings (int*, uint*, float*, bool*), use FloatGetter
	// This is the base implementation in modbus plugin
	return callGetter(p.(plugin.FloatGetter).FloatGetter())
}

// callGetter calls a getter function and returns the result
func callGetter[T any](getterFn func() (T, error), err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return getterFn()
}

// applyCast applies optional type casting
func applyCast(value any, castType string) any {
	switch strings.ToLower(castType) {
	case "int":
		return cast.ToInt64(value)
	case "float":
		return cast.ToFloat64(value)
	case "string":
		return cast.ToString(value)
	default:
		return value
	}
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
