package modbus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSharedSettings ensures the largest delay and timeout wins for all
// connections sharing the same physical connection
func TestSharedSettings(t *testing.T) {
	ctx := t.Context()
	uri := "localhost:15020"

	c1, err := Settings{URI: uri, ID: 1, Delay: 2 * time.Second, Timeout: time.Second}.Connection(ctx)
	require.NoError(t, err)

	c2, err := Settings{URI: uri, ID: 2, Delay: time.Second, Timeout: 3 * time.Second}.Connection(ctx)
	require.NoError(t, err)

	// unset settings don't reset the shared values
	c3, err := Settings{URI: uri, ID: 3}.Connection(ctx)
	require.NoError(t, err)

	require.Same(t, c1.physical, c2.physical)
	require.Same(t, c1.physical, c3.physical)
	require.Same(t, c1.physical, c1.Clone(4).physical)

	for _, c := range []*Connection{c1, c2, c3} {
		require.Equal(t, 2*time.Second, c.physical.getDelay())
		require.Equal(t, 3*time.Second, c.physical.timeout)
	}

	// timeout has been applied to the physical connection
	require.Equal(t, 3*time.Second, c1.physical.Connection.Timeout(3*time.Second))
}

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
