package modbussvc

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
	"github.com/fatih/structs"
	"github.com/spf13/cast"
)

var log = util.NewLogger("modbus")

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
	Result          string
	Scale           float64 // scaling factor
	Cast            string  // type cast (int, float, string)
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
	if query.URI == "" || query.Address == 0 {
		jsonError(w, http.StatusBadRequest, fmt.Errorf("uri and address parameters are required"))
		return
	}

	// Create cache key from URI and register address
	cacheKey := fmt.Sprintf("%s:%d", query.URI, query.Address)

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
		log.DEBUG.Printf("Failed to read register %d: %v", query.Address, err)
		jsonWrite(w, []string{}) // Return empty array on error
		return
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
	p, err := plugin.NewModbusFromConfig(ctx, structs.Map(query))
	if err != nil {
		return nil, fmt.Errorf("failed to create modbus plugin: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid result type: %s", query.Result)
		}
	}()

	switch strings.ToLower(query.Result) {
	case "float":
		return p.(plugin.FloatGetter).FloatGetter()
	case "int":
		return p.(plugin.IntGetter).IntGetter()
	case "bool":
		return p.(plugin.BoolGetter).BoolGetter()
	case "string":
		return p.(plugin.StringGetter).StringGetter()
	default:
		return nil, fmt.Errorf("plugin does not implement any supported getter interface")
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
