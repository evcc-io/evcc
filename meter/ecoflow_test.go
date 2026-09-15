package meter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEcoflowReserveLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit float64
		lo    float64
		want  float64
	}{
		{name: "above discharge limit", limit: 50, lo: 7, want: 50},
		{name: "at required distance", limit: 10, lo: 7, want: 10},
		{name: "below required distance", limit: 5, lo: 7, want: 10},
		{name: "discharge limit raised", limit: 10, lo: 15, want: 18},
		{name: "charge mode", limit: 100, lo: 7, want: 100},
		{name: "distance exceeds full reserve", limit: 10, lo: 99, want: 100},
		{name: "zero discharge limit", limit: 0, lo: 0, want: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ecoflowReserveLimit(tc.limit, tc.lo))
		})
	}
}
