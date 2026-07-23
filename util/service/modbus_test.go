package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

func TestGetParams_DirectURI(t *testing.T) {
	// Verify that direct URI parameter works
	req := httptest.NewRequest("GET", "/read?uri=192.168.1.1:502&address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestGetParams_WithScale(t *testing.T) {
	// Test with scale parameter
	req := httptest.NewRequest("GET", "/read?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestGetParams_WithResultType(t *testing.T) {
	// Test with resulttype parameter
	req := httptest.NewRequest("GET", "/read?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&resulttype=int", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestGetParams_CompleteRequest(t *testing.T) {
	// Test complete request with all parameters
	req := httptest.NewRequest("GET", "/read?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001&resulttype=int", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestGetParams_RS485Serial(t *testing.T) {
	// Test RS485 serial connection with device parameter
	req := httptest.NewRequest("GET", "/read?device=/dev/ttyUSB0&baudrate=9600&comset=8N1&id=1&address=1068&type=holding&encoding=float32s&scale=0.001", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestGetParams_RS485Serial_WithResultType(t *testing.T) {
	// Test RS485 serial with resulttype parameter
	req := httptest.NewRequest("GET", "/read?device=/dev/ttyUSB0&baudrate=19200&comset=8N1&id=1&address=100&type=holding&encoding=uint16&resulttype=int", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestGetParams_MissingConnection(t *testing.T) {
	// Test that either uri or device is required
	req := httptest.NewRequest("GET", "/read?id=1&address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	// Should return 400 error
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "uri or device")
}

func TestGetParams_MissingAddress(t *testing.T) {
	// Test that address parameter is required
	req := httptest.NewRequest("GET", "/read?uri=192.168.1.1:502&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "address")
}

func TestGetParams_AddressZero(t *testing.T) {
	// Test that address=0 is valid (not treated as missing)
	req := httptest.NewRequest("GET", "/read?uri=192.168.1.1:502&address=0&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	modbusRead(w, req)

	// Should NOT return 400 - address 0 is valid, will fail at connection
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestApplyCast(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		castType string
		expected any
	}{
		// Int conversions
		{"float to int", 42.7, "int", int64(42)},
		{"string to int", "42", "int", int64(42)},
		{"int to int", 42, "int", int64(42)},
		{"negative float to int", -42.9, "int", int64(-42)},

		// Float conversions
		{"int to float", 42, "float", float64(42.0)},
		{"string to float", "42.7", "float", float64(42.7)},
		{"float to float", 42.7, "float", float64(42.7)},

		// String conversions
		{"int to string", 42, "string", "42"},
		{"float to string", 42.7, "string", "42.7"},
		{"string to string", "hello", "string", "hello"},

		// Unknown/empty type (should return original)
		{"unknown type", 42, "unknown", 42},
		{"empty type", 42, "", 42},
		{"nil value", nil, "int", int64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.value, tt.castType)
			assert.Equal(t, cast.ToString(tt.expected), result)
		})
	}
}
