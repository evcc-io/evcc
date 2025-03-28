package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
