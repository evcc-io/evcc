package modbus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePoint(t *testing.T) {
	tc := []struct {
		in  string
		ops SunSpecOperation
	}{
		{"103:W", SunSpecOperation{103, 0, "W"}},
		{"802:1:V", SunSpecOperation{802, 1, "V"}},
	}

	for _, tc := range tc {
		t.Log(tc)

		ops, err := ParsePoint(tc.in)
		require.NoError(t, err)
		require.Equal(t, tc.ops, ops)
	}
}
