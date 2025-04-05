package server

import (
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSquashedSettings(t *testing.T) {
	old := globalconfig.Mqtt{
		Config: mqtt.Config{
			Broker: "host",
			User:   "user",
		},
		Topic: "test",
	}
	{
		new := old
		new.User = masked

		require.NoError(t, mergeSettings(old, &new))
		assert.Equal(t, "user", new.User)
	}
	{
		new := old
		new.User = "new"

		require.NoError(t, mergeSettings(old, &new))
		assert.Equal(t, "new", new.User)
	}
}

type testStruct struct {
	Field1 string
	Field2 int
}

func TestMergeSettings(t *testing.T) {
	tests := []struct {
		old           any
		new, expected *testStruct
	}{
		{
			old:      nil,
			new:      &testStruct{"newValue1", 42},
			expected: &testStruct{"newValue1", 42},
		},
		{
			old:      &testStruct{"oldValue1", 24},
			new:      &testStruct{"newValue1", 42},
			expected: &testStruct{"newValue1", 42},
		},
		{
			old:      &testStruct{"oldValue1", 24},
			new:      &testStruct{masked, 42},
			expected: &testStruct{"oldValue1", 42},
		},
	}

	for _, tc := range tests {
		mergeSettings(tc.old, tc.new)
		assert.Equal(t, tc.expected, tc.new)
	}
}
