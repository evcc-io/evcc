package server

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/evcc-io/evcc/core"
	"github.com/stretchr/testify/assert"
)

func TestErrorStatus(t *testing.T) {
	// settings the optimizer decides on are rejected as conflict, not bad request
	assert.Equal(t, http.StatusConflict, errorStatus(core.ErrOptimizerAutomatic))
	assert.Equal(t, http.StatusConflict, errorStatus(fmt.Errorf("smart cost limit: %w", core.ErrOptimizerAutomatic)))
	assert.Equal(t, http.StatusBadRequest, errorStatus(errors.New("boom")))
}
