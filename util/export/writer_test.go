package export

import (
	"testing"
	"time"
)

func TestFormatValue_NilPointer(t *testing.T) {
	var nilPtr *float64
	if got := formatValue(nil, nilPtr, 3); got != "" {
		t.Errorf("Expected empty string for nil pointer, got: %s", got)
	}
}

func TestFormatValue_ZeroTime(t *testing.T) {
	if got := formatValue(nil, time.Time{}, 3); got != "" {
		t.Errorf("Expected empty string for zero time, got: %s", got)
	}
}
