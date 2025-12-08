package modbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
	gridx "github.com/grid-x/modbus"
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

// ParamValue represents a single parameter value read from modbus
type ParamValue struct {
	Value any    `json:"value"`
	Unit  string `json:"unit,omitempty"`
	Error string `json:"error,omitempty"`
}

// getParams reads parameter values from a device based on template configuration
// If 'param' query parameter is provided, returns single value as array (for UI)
// If 'param' is not provided, returns all values as object (for debugging)
func getParams(w http.ResponseWriter, req *http.Request) {
	// Extract query parameters
	templateName := req.URL.Query().Get("template")
	uri := req.URL.Query().Get("uri")
	host := req.URL.Query().Get("host")
	port := req.URL.Query().Get("port")
	idStr := req.URL.Query().Get("id")
	paramName := req.URL.Query().Get("param") // Optional: specific parameter name

	// Validate required parameters
	if templateName == "" {
		jsonError(w, http.StatusBadRequest, errors.New("missing template parameter"))
		return
	}

	// Build URI from host and port if uri not directly provided
	if uri == "" {
		if host == "" {
			jsonError(w, http.StatusBadRequest, errors.New("missing uri or host parameter"))
			return
		}
		// Default port if not specified
		if port == "" {
			port = DefaultModbusPort
		}
		uri = fmt.Sprintf("%s:%s", host, port)
	}

	// Parse device ID
	var modbusSettings Settings
	modbusSettings.URI = uri
	if idStr != "" {
		if _, err := fmt.Sscanf(idStr, "%d", &modbusSettings.ID); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid id parameter: %w", err))
			return
		}
	}

	// Load template
	tmpl, err := templates.ByName(templates.Meter, templateName)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Errorf("template '%s' not found: %w", templateName, err))
		return
	}

	// Read parameters with context timeout
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	result := make(map[string]ParamValue)

	// Iterate over template parameters and read those with register configuration
	for _, param := range tmpl.Params {
		// Check if parameter has register configuration in properties
		if param.Properties == nil {
			continue
		}

		registerCfg, hasRegister := param.Properties["register"]
		if !hasRegister {
			continue
		}

		// Decode register configuration
		var reg Register
		if err := util.DecodeOther(registerCfg, &reg); err != nil {
			result[param.Name] = ParamValue{
				Error: fmt.Sprintf("failed to decode register config: %v", err),
			}
			continue
		}

		// Read from modbus
		value, err := readRegister(ctx, modbusSettings, reg)
		if err != nil {
			result[param.Name] = ParamValue{
				Error: err.Error(),
			}
			log.DEBUG.Printf("Failed to read %s: %v", param.Name, err)
			continue
		}

		// Get scale factor if present
		scale := 1.0
		if scaleCfg, ok := param.Properties["scale"]; ok {
			if s, ok := scaleCfg.(float64); ok {
				scale = s
			}
		}

		// Apply scale
		scaledValue := value * scale

		// Apply cast if present
		var finalValue any = scaledValue
		if castCfg, ok := param.Properties["cast"]; ok {
			if castType, ok := castCfg.(string); ok {
				switch castType {
				case "int":
					finalValue = int64(scaledValue + 0.5) // Round to nearest
				case "float":
					finalValue = scaledValue
				case "string":
					finalValue = fmt.Sprintf("%v", scaledValue)
				}
			}
		}

		// Store result with unit from param description
		result[param.Name] = ParamValue{
			Value: finalValue,
			Unit:  param.Unit,
		}

		log.DEBUG.Printf("Read %s: %v %s", param.Name, finalValue, param.Unit)
	}

	// If specific parameter requested, return as string array (for UI compatibility)
	if paramName != "" {
		if paramValue, ok := result[paramName]; ok {
			// If there was an error reading the parameter, return empty array
			if paramValue.Error != "" {
				jsonWrite(w, []string{})
				return
			}
			// Return value as string array
			valueStr := fmt.Sprintf("%v", paramValue.Value)
			jsonWrite(w, []string{valueStr})
			return
		}
		// Parameter not found or error
		jsonWrite(w, []string{})
		return
	}

	// Return all results as object (for debugging/batch requests)
	jsonWrite(w, result)
}

// readRegister reads a single modbus register
func readRegister(ctx context.Context, settings Settings, reg Register) (float64, error) {
	if err := reg.Error(); err != nil {
		return 0, fmt.Errorf("invalid register config: %w", err)
	}

	// Create temporary modbus connection
	Lock()
	defer Unlock()

	conn, err := NewConnection(ctx, settings.URI, settings.Device,
		settings.Comset, settings.Baudrate, settings.Protocol(), settings.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to create modbus connection: %w", err)
	}

	// Get register operation
	funcCode, err := reg.FuncCode()
	if err != nil {
		return 0, fmt.Errorf("failed to get function code: %w", err)
	}

	length, err := reg.Length()
	if err != nil {
		return 0, fmt.Errorf("failed to get register length: %w", err)
	}

	// Get decoder
	decode, err := reg.DecodeFunc()
	if err != nil {
		return 0, fmt.Errorf("failed to get decode function: %w", err)
	}

	// Read from register
	var bytes []byte
	switch funcCode {
	case gridx.FuncCodeReadHoldingRegisters:
		bytes, err = conn.ReadHoldingRegisters(reg.Address, length)
	case gridx.FuncCodeReadInputRegisters:
		bytes, err = conn.ReadInputRegisters(reg.Address, length)
	default:
		return 0, fmt.Errorf("unsupported function code: %d (only holding and input registers supported)", funcCode)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to read register %d: %w", reg.Address, err)
	}

	// Decode value
	value := decode(bytes)
	return value, nil
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
