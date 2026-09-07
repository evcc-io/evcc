package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSunspecRead_MissingValue(t *testing.T) {
	// point value is required
	req := httptest.NewRequest("GET", "/read?uri=192.168.1.1:502&id=1", nil)
	w := httptest.NewRecorder()

	sunspecRead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSunspecRead_MissingConnection(t *testing.T) {
	// uri or device is required
	req := httptest.NewRequest("GET", "/read?id=1&value=124:0:InOutWRte_RvrtTms", nil)
	w := httptest.NewRecorder()

	sunspecRead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
