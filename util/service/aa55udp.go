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
	"github.com/spf13/cast"
)

// AA55Query holds the aa55udp connection and register read parameters
type AA55Query struct {
	Host            string
	Id              int
	modbus.Register `mapstructure:",squash"`
	Scale           float64 // scaling factor
	ResultType      string  // type cast (int, float, string)
}

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /read", aa55udpRead)

	service.Register("aa55udp", mux)
}

// aa55udpRead reads a register value from a GoodWe inverter via AA55-over-UDP.
// It mirrors modbusRead and is used to derive config-time defaults (e.g. rated power).
func aa55udpRead(w http.ResponseWriter, req *http.Request) {
	cc := make(map[string]any)
	for k := range req.URL.Query() {
		cc[k] = req.URL.Query().Get(k)
	}

	query := AA55Query{
		Id:    247,
		Scale: 1.0,
	}

	if err := util.DecodeOther(cc, &query); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	if query.Host == "" || cc["address"] == nil {
		jsonError(w, http.StatusBadRequest, fmt.Errorf("host and address parameters are required"))
		return
	}

	cacheKey := fmt.Sprintf("aa55:%s:%d", query.Host, query.Address)

	mu.RLock()
	if entry, ok := cache[cacheKey]; ok && time.Since(entry.timestamp) < cacheTTL {
		mu.RUnlock()
		jsonWrite(w, []string{cast.ToString(entry.value)})
		return
	}
	mu.RUnlock()

	// background context so the connection isn't tied to the HTTP request lifecycle
	value, err := readAA55Value(context.TODO(), query)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	res := toString(value, query.ResultType)

	mu.Lock()
	cache[cacheKey] = cacheEntry{value: res, timestamp: time.Now()}
	mu.Unlock()

	jsonWrite(w, []string{res})
}

// readAA55Value reads a register value by reusing the aa55udp plugin
func readAA55Value(ctx context.Context, query AA55Query) (res any, err error) {
	cfg := map[string]any{
		"host":     query.Host,
		"id":       query.Id,
		"register": query.Register,
		"scale":    query.Scale,
	}

	p, err := plugin.NewAA55UDPFromConfig(ctx, cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to create aa55udp plugin: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("read failed: %v", r)
		}
	}()

	g, err := p.(plugin.FloatGetter).FloatGetter()
	if err != nil {
		return nil, err
	}

	return g()
}
