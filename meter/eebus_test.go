package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/util/test"
)

func TestEEBus(t *testing.T) {
	acceptable := []string{
		"eebus not configured",
	}

	values := map[string]any{
		"ski":     "test-ski",
		"ip":      "192.0.2.2",
		"usage":   "grid",
		"timeout": "10s",
	}

	if _, err := NewFromConfig(context.TODO(), "eebus", values); err != nil && !test.Acceptable(err, acceptable) {
		t.Error(err)
	}
}
