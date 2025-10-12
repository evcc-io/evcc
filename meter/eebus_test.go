package meter

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/util/test"
)

func TestEEBus(t *testing.T) {
	acceptable := []string{
		"eebus is not configured yet. check config regarding cert, keys etc.",
	}

	values := map[string]any{
		"ski":     "22dd0b546beaa6a720302119c87fc5e0e7ae2e07",
		"ip":      "192.0.2.2",
		"usage":   "grid",
		"timeout": "10s",
	}

	if _, err := NewFromConfig(context.TODO(), "eebus", values); err != nil && !test.Acceptable(err, acceptable) {
		t.Error(err)
	}
}
