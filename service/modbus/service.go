package modbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

var log = util.NewLogger("modbus")

const (
	// DefaultModbusPort is the standard Modbus TCP port
	DefaultModbusPort = "502"
	// DefaultModbusID is the default Modbus device ID
	DefaultModbusID = 1
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /params", getParams)

	service.Register("modbus", mux)
}

// getParams reads a parameter value from a device based on URL parameters
// Returns single value as array (for UI compatibility)
func getParams(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	q := req.URL.Query()

	// Build modbus settings from query parameters
	settings, err := settingsFromQuery(q)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// Parse register configuration from query parameters
	reg, scale, cast, err := parseRegisterFromQuery(q)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// Read value from modbus using plugin (zero duplication!)
	value, err := readRegisterValue(ctx, settings, reg, scale)
	if err != nil {
		log.DEBUG.Printf("Failed to read register %d: %v", reg.Address, err)
		jsonWrite(w, []string{}) // Return empty array on error
		return
	}

	// Apply cast if specified
	finalValue := applyCast(value, cast)

	// Return value as string array (for UI compatibility)
	valueStr := fmt.Sprintf("%v", finalValue)
	jsonWrite(w, []string{valueStr})
}

// settingsFromQuery builds modbus settings from URL query parameters
func settingsFromQuery(q url.Values) (modbus.Settings, error) {
	var settings modbus.Settings

	// Build URI from host and port, or use uri directly
	uri := q.Get("uri")
	if uri == "" {
		host := q.Get("host")
		if host == "" {
			return settings, errors.New("missing uri or host parameter")
		}
		port := q.Get("port")
		if port == "" {
			port = DefaultModbusPort
		}
		uri = fmt.Sprintf("%s:%s", host, port)
	}
	settings.URI = uri

	// Parse device ID
	idStr := q.Get("id")
	if idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return settings, fmt.Errorf("invalid id parameter: %w", err)
		}
		settings.ID = uint8(id)
	}

	return settings, nil
}

// parseRegisterFromQuery extracts register configuration from URL query parameters
func parseRegisterFromQuery(q url.Values) (modbus.Register, float64, string, error) {
	var reg modbus.Register
	scale := 1.0

	// Required parameters
	addressStr := q.Get("address")
	if addressStr == "" {
		return reg, 0, "", errors.New("missing address parameter")
	}
	address, err := strconv.ParseUint(addressStr, 10, 16)
	if err != nil {
		return reg, 0, "", fmt.Errorf("invalid address parameter: %w", err)
	}
	reg.Address = uint16(address)

	regType := q.Get("type")
	if regType == "" {
		return reg, 0, "", errors.New("missing type parameter (holding/input)")
	}
	reg.Type = regType

	encoding := q.Get("encoding")
	if encoding == "" {
		return reg, 0, "", errors.New("missing encoding parameter")
	}
	reg.Encoding = encoding

	// Optional parameters
	if scaleStr := q.Get("scale"); scaleStr != "" {
		scale, err = strconv.ParseFloat(scaleStr, 64)
		if err != nil {
			return reg, 0, "", fmt.Errorf("invalid scale parameter: %w", err)
		}
	}

	cast := q.Get("cast")

	return reg, scale, cast, nil
}

// readRegisterValue reads a modbus register value by reusing the modbus plugin
// This completely eliminates code duplication with plugin/modbus.go
func readRegisterValue(ctx context.Context, settings modbus.Settings, reg modbus.Register, scale float64) (float64, error) {
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
		return 0, fmt.Errorf("failed to create modbus plugin: %w", err)
	}

	// Get float getter
	getter, err := p.(plugin.FloatGetter).FloatGetter()
	if err != nil {
		return 0, fmt.Errorf("failed to get float getter: %w", err)
	}

	// Read value once
	return getter()
}

// applyCast applies type casting to the value
func applyCast(value float64, cast string) any {
	switch cast {
	case "int":
		return int64(value + 0.5) // Round to nearest
	case "float":
		return value
	case "string":
		return fmt.Sprintf("%v", value)
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
