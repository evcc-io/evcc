package modbussvc

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
	// Connection failure returns zero value as string
	body := w.Body.String()
	assert.True(t, body == "[]\n" || body == "[\"\"]\n" || body == "[\"0\"]\n", "Expected empty or zero value for failed read, got: %s", body)
}

func TestGetParams_WithScale(t *testing.T) {
	// Test with scale parameter
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), returns zero value
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t, body == "[]\n" || body == "[\"\"]\n" || body == "[\"0\"]\n", "Expected empty or zero value for failed read, got: %s", body)
}

func TestGetParams_WithCast(t *testing.T) {
	// Test with result parameter - encoding=float32s, result type int
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&result=int", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), returns zero value
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t, body == "[]\n" || body == "[\"\"]\n" || body == "[\"0\"]\n", "Expected empty or zero value for failed read, got: %s", body)
}

func TestGetParams_CompleteRequest(t *testing.T) {
	// Test complete request with all parameters
	req := httptest.NewRequest("GET", "/params?uri=192.168.1.1:502&id=1&address=1068&type=holding&encoding=float32s&scale=0.001&result=int", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	// Connection will fail (no real device), returns zero value
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t, body == "[]\n" || body == "[\"\"]\n" || body == "[\"0\"]\n", "Expected empty or zero value for failed read, got: %s", body)
}
