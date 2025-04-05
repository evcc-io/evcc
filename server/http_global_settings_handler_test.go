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
		new.User = "***"

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

func TestMergeSettings(t *testing.T) {
	tests := []struct {
		old      any
		new      *RedactedStruct
		expected *RedactedStruct
	}{
		{
			old:      nil,
			new:      &RedactedStruct{"newValue1", 42},
			expected: &RedactedStruct{"newValue1", 42},
		},
		{
			old:      &TestStruct{"oldValue1", 24},
			new:      &RedactedStruct{"newValue1", 42},
			expected: &RedactedStruct{"newValue1", 42},
		},
		{
			old:      &RedactedStruct{"oldValue1", 24},
			new:      &RedactedStruct{"redacted", 42},
			expected: &RedactedStruct{"oldValue1", 42},
		},
		{
			old:      &RedactedStruct{"oldValue1", 24},
			new:      &RedactedStruct{"newValue1", 42},
			expected: &RedactedStruct{"newValue1", 42},
		},
	}

	for _, tc := range tests {
		mergeSettings(tc.old, tc.new)
		assert.Equal(t, tc.expected.Field1, tc.new.Field1)
		assert.Equal(t, tc.expected.Field2, tc.new.Field2)
	}
}

type TestStruct struct {
	Field1 string
	Field2 int
}

type RedactedStruct struct {
	Field1 string
	Field2 int
}

func (t *RedactedStruct) Redacted() any {
	return struct {
		Field1 string
		Field2 int
	}{
		Field1: "redacted",
		Field2: t.Field2,
	}
}
