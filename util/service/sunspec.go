package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/fatih/structs"
	"github.com/spf13/cast"
)

// SunSpecQuery combines modbus settings, sunspec point and additional parameters
type SunSpecQuery struct {
	modbus.Settings `mapstructure:",squash"`
	Value           string  // sunspec point, e.g. 124:0:InOutWRte_RvrtTms
	Scale           float64 // scaling factor
	ResultType      string  // type cast (int, float, string)
}

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /read", sunspecRead)

	service.Register("sunspec", mux)
}

// sunspecRead reads a sunspec point value from a device based on URL parameters
// Returns single value as array (for UI compatibility)
func sunspecRead(w http.ResponseWriter, req *http.Request) {
	// Convert URL query parameters to map for decoding
	cc := make(map[string]any)
	for k := range req.URL.Query() {
		cc[k] = req.URL.Query().Get(k)
	}

	query := SunSpecQuery{
		Scale: 1.0,
	}

	if err := util.DecodeOther(cc, &query); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// Validate required parameters
	if (query.URI == "" && query.Device == "") || query.Value == "" {
		jsonError(w, http.StatusBadRequest, fmt.Errorf("uri or device and value parameters are required"))
		return
	}

	// Cache per exact request: distinct id/point/scale/resulttype must not share a converted value
	cacheKey := "sunspec/" + req.URL.Query().Encode()

	mu.RLock()
	if entry, ok := cache[cacheKey]; ok && time.Since(entry.timestamp) < cacheTTL {
		mu.RUnlock()
		jsonWrite(w, []string{cast.ToString(entry.value)})
		return
	}
	mu.RUnlock()

	// Use background context so connection isn't tied to HTTP request lifecycle
	value, err := readPointValue(context.TODO(), query)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	res := toString(value, query.ResultType)

	mu.Lock()
	cache[cacheKey] = cacheEntry{
		value:     res,
		timestamp: time.Now(),
	}
	mu.Unlock()

	jsonWrite(w, []string{res})
}

// readPointValue reads a sunspec point value by reusing the sunspec plugin
func readPointValue(ctx context.Context, query SunSpecQuery) (any, error) {
	// Convert Settings to map (plugin expects Settings fields at top level)
	cfg := structs.Map(query.Settings)

	cfg["value"] = []string{query.Value}
	cfg["scale"] = query.Scale

	p, err := plugin.NewModbusSunspecFromConfig(ctx, cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to create sunspec plugin: %w", err)
	}

	g, err := p.(plugin.FloatGetter).FloatGetter()
	if err != nil {
		return nil, err
	}

	var res float64
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("read failed: %v", r)
			}
		}()

		res, err = g()
	}()
	if err != nil {
		return nil, err
	}

	return res, nil
}
