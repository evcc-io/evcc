package charger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDepowStep(t *testing.T) {
	for _, tc := range []struct {
		current, expected int64
	}{
		{5, 6}, {6, 6}, {7, 6}, {8, 8}, {9, 8}, {12, 10}, {13, 13}, {15, 13}, {16, 16}, {32, 16},
	} {
		assert.Equal(t, tc.expected, depowStep(depowDefaultSteps, tc.current), tc.current)
	}
}
