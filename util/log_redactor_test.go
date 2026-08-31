package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRotatingSlot(t *testing.T) {
	var r Redactor
	r.Redact("static")
	slot := r.RotatingSlot()

	assert.Equal(t, "*** token", string(r.redacted([]byte("static token"))))

	slot("first")
	assert.Equal(t, "*** ***", string(r.redacted([]byte("static first"))))

	// updating the slot replaces the previous value instead of appending
	slot("second")
	assert.Equal(t, "*** ***", string(r.redacted([]byte("static second"))))
	assert.Equal(t, "*** first", string(r.redacted([]byte("static first"))))
	assert.Len(t, r.rotating, 1)
}
