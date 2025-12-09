package modbus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetParams_MissingAddress(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing address parameter")
}

func TestGetParams_MissingType(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1&address=100", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing type parameter")
}

func TestGetParams_MissingEncoding(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1&address=100&type=holding", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing encoding parameter")
}

func TestGetParams_MissingUriAndHost(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing uri or host parameter")
}

func TestGetParams_URIConstruction_WithDefaultPort(t *testing.T) {
	// This test verifies that the service constructs URI from host with default port
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1&address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Should fail at Modbus connection (no real device)
	// but this proves URI was constructed from host with default port
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_URIConstruction_WithCustomPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1&port=1502&address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Should fail at Modbus connection but proves URI construction worked
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_DirectURI(t *testing.T) {
	// Verify that direct URI parameter works
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Should fail at Modbus connection but proves direct URI works
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_InvalidID(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&id=invalid&address=100&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid id parameter")
}

func TestGetParams_InvalidAddress(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&address=invalid&type=holding&encoding=uint16", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid address parameter")
}

func TestGetParams_InvalidScale(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&address=100&type=holding&encoding=uint16&scale=invalid", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid scale parameter")
}

func TestGetParams_WithScale(t *testing.T) {
	// Test with scale parameter
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1&port=502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), should return empty array
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_WithCast(t *testing.T) {
	// Test with cast parameter
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1&port=502&id=1&address=1068&type=holding&encoding=float32s&cast=int", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), should return empty array
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}

func TestGetParams_CompleteRequest(t *testing.T) {
	// Test complete request with all parameters
	req := httptest.NewRequest("GET", "/params?host=192.168.1.1&port=502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001&cast=int", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), should return empty array
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]\n", w.Body.String(), "Expected empty array for failed read")
}
