package export

import (
	"testing"
	"time"
)

func TestNormalizeValue_NilPointer(t *testing.T) {
	var nilPtr *float64
	if got := normalizeValue(nilPtr, 3); got != nil {
		t.Errorf("Expected nil for nil pointer, got: %v", got)
	}
}

func TestNormalizeValue_ZeroTime(t *testing.T) {
	if got := normalizeValue(time.Time{}, 3); got != nil {
		t.Errorf("Expected nil for zero time, got: %v", got)
	}
}

func TestNormalizeValue_Rounding(t *testing.T) {
	if got := normalizeValue(1.23456789, 4); got != 1.2346 {
		t.Errorf("Expected 1.2346, got: %v", got)
	}
	if got := normalizeValue(42.5, 0); got != 43.0 {
		t.Errorf("Expected 43, got: %v", got)
	}
}
