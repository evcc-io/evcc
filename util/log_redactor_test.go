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

	slot("access", "refresh")
	assert.Equal(t, "*** *** ***", string(r.redacted([]byte("static access refresh"))))

	// updating the slot replaces the previous values instead of appending
	slot("second")
	assert.Equal(t, "*** ***", string(r.redacted([]byte("static second"))))
	assert.Equal(t, "*** access refresh", string(r.redacted([]byte("static access refresh"))))
	assert.Len(t, r.rotating, 1)
}

func TestRotatingSlotLimit(t *testing.T) {
	var r Redactor

	for range maxRotatingSlots + 1 {
		r.RotatingSlot()
	}

	assert.Len(t, r.rotating, maxRotatingSlots)
}
