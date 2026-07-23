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

func TestSettingsProtocol(t *testing.T) {
	tc := []struct {
		Settings
		res Protocol
	}{
		{Settings{UDP: true}, Udp},
		{Settings{RTU: new(true)}, Rtu},
		{Settings{Device: "foo"}, Rtu},
		{Settings{URI: "foo"}, Tcp},
		{Settings{}, Tcp},
	}

	for _, tc := range tc {
		require.Equal(t, tc.res, tc.Protocol(), tc)
	}
}
