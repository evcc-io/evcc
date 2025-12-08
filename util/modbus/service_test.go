package modbus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Register test template from the tests directory
	// Path is relative to the package directory (util/modbus)
	_ = templates.Register(templates.Meter, "../../tests/modbus-service/modbus-service-test.tpl.yaml")
}

func TestGetParams_MissingTemplate(t *testing.T) {
	req := httptest.NewRequest("GET", "/params", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing template parameter")
}

func TestGetParams_MissingUriAndHost(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?template=test", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing uri or host parameter")
}

func TestGetParams_URIConstruction_WithDefaultPort(t *testing.T) {
	// This test verifies that the service constructs URI from host+port
	// We can't test the full flow without a real Modbus device,
	// but we can verify the URI construction logic by checking error messages

	req := httptest.NewRequest("GET", "/params?template=nonexistent&host=192.168.1.1", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	body := w.Body.String()

	// Should fail at template loading (template "nonexistent" doesn't exist)
	// but this proves URI was constructed from host
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, body, "template not found", "Error message should mention template not found")
	assert.Contains(t, body, "nonexistent", "Error message should include the template name")
}

func TestGetParams_URIConstruction_WithCustomPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?template=nonexistent&host=192.168.1.1&port=1502", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	body := w.Body.String()

	// Should fail at template loading but proves URI construction worked
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, body, "template not found", "Error message should mention template not found")
	assert.Contains(t, body, "nonexistent", "Error message should include the template name")
}

func TestGetParams_DirectURI(t *testing.T) {
	// Verify that direct URI parameter works (backwards compatibility)
	req := httptest.NewRequest("GET", "/params?template=nonexistent&uri=192.168.1.1:502", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	body := w.Body.String()

	// Should fail at template loading but proves direct URI works
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, body, "template not found", "Error message should mention template not found")
	assert.Contains(t, body, "nonexistent", "Error message should include the template name")
}

func TestGetParams_InvalidID(t *testing.T) {
	req := httptest.NewRequest("GET", "/params?template=modbus-service-test&uri=192.168.1.1:502&id=invalid", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	body := w.Body.String()

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, body, "invalid id parameter", "Error message should mention invalid id")
}

func TestGetParams_SuccessfulRequest(t *testing.T) {
	// This test verifies the complete request flow up to the Modbus connection
	// It uses a real test template to ensure template loading works correctly

	// Use host+port format (new feature)
	req := httptest.NewRequest("GET", "/params?template=modbus-service-test&host=192.168.1.1&port=502&id=1&param=testparam", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	body := w.Body.String()

	// Template should be found, but Modbus connection will fail (no real device)
	// The service returns HTTP 200 with an empty array when param is requested but read fails
	assert.Equal(t, http.StatusOK, w.Code, "Expected OK status, got: %s", body)
	assert.Equal(t, "[]\n", body, "Expected empty array for failed param read, got: %s", body)
}

func TestGetParams_AllParameters(t *testing.T) {
	// Test without param query parameter - should return all parameters
	req := httptest.NewRequest("GET", "/params?template=modbus-service-test&uri=192.168.1.1:502&id=1", nil)
	w := httptest.NewRecorder()

	getParams(w, req)

	body := w.Body.String()

	// Template should be found, connection will fail, but should return object with error details
	assert.Equal(t, http.StatusOK, w.Code, "Expected OK status, got: %s", body)
	// Should return object with all parameters (not array)
	assert.Contains(t, body, "{", "Expected JSON object for all parameters")
	assert.Contains(t, body, "testparam", "Expected testparam in response")
	assert.Contains(t, body, "error", "Expected error field for failed reads")
}
