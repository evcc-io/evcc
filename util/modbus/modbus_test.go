package modbus_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/evcc-io/evcc/util/modbus"
)

func TestParsePoint(t *testing.T) {
	tc := []struct {
		in  string
		ops modbus.SunSpecOperation
	}{
		{in: "103:W", ops: modbus.SunSpecOperation{Model: 103, Point: "W"}},
		{in: "802:1:V", ops: modbus.SunSpecOperation{Model: 802, Block: 1, Point: "V"}},
	}

	for _, tc := range tc {
		t.Log(tc)

		ops, err := modbus.ParsePoint(tc.in)
		require.NoError(t, err)
		require.Equal(t, tc.ops, ops)
	}
}
