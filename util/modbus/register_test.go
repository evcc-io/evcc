package modbus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLength(t *testing.T) {
	tc := []struct {
		value string
		want  uint16
	}{
		{"bool8", 1},
		{"int16", 1},
		{"float32", 2},
		{"uint64s", 4},
	}

	for _, tc := range tc {
		res, err := Register{Encoding: tc.value}.Length()
		require.NoError(t, err, tc)
		require.Equal(t, tc.want, res, tc)
	}
}
