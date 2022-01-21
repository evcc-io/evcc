package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProgress(t *testing.T) {
	p := NewProgress(0, 10)

	tc := []struct {
		value float64
		res   bool
	}{
		{-1, false},
		{0, true},
		{1, false},
		{5, false},
		{10, true},
		{15, false},
		{25, true},
		{30, true},
		{60, true},
		{65, false},
		{70, true},
	}

	for _, tc := range tc {
		require.Equal(t, tc.res, p.NextStep(tc.value), fmt.Sprintf("%.0f%%", tc.value))
	}
}
