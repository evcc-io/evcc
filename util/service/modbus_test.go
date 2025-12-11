package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetParams_DirectURI(t *testing.T) {
	// Verify that direct URI parameter works
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Should fail at Modbus connection but proves direct URI works
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_WithScale(t *testing.T) {
	// Test with scale parameter
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), should return empty array
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_WithResultType(t *testing.T) {
	// Test with resulttype parameter
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&resulttype=int", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), should return empty array
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_CompleteRequest(t *testing.T) {
	// Test complete request with all parameters
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001&resulttype=int", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), should return empty array
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}
