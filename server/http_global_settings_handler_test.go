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
		new      any
		expected any
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
		{
			old:      &NestedStruct{RedactedStruct{"oldValue1", 35}, 24},
			new:      &NestedStruct{RedactedStruct{"newValue1", 45}, 42},
			expected: &NestedStruct{RedactedStruct{"newValue1", 45}, 42},
		},
		{
			old:      &NestedStruct{RedactedStruct{"oldValue1", 35}, 24},
			new:      &NestedStruct{RedactedStruct{"redacted", 45}, 0},
			expected: &NestedStruct{RedactedStruct{"oldValue1", 45}, 24},
		},
	}

	for _, tc := range tests {
		mergeSettings(tc.old, tc.new)
		switch expected := tc.expected.(type) {
		case *RedactedStruct:
			assert.Equal(t, expected.Field1, tc.new.(*RedactedStruct).Field1)
			assert.Equal(t, expected.Field2, tc.new.(*RedactedStruct).Field2)
		case *NestedStruct:
			assert.Equal(t, expected.Field1.Field1, tc.new.(*NestedStruct).Field1.Field1)
			assert.Equal(t, expected.Field1.Field2, tc.new.(*NestedStruct).Field1.Field2)
			assert.Equal(t, expected.Field2, tc.new.(*NestedStruct).Field2)
		}
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

type NestedStruct struct {
	Field1 RedactedStruct
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

func (t *NestedStruct) Redacted() any {
	return struct {
		Field1 RedactedStruct
		Field2 int
	}{
		Field1: t.Field1.Redacted().(struct {
			Field1 string
			Field2 int
		}),
		Field2: 0,
	}
}
