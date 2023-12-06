package server

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	now := time.Now()

	tc := []struct {
		in  interface{}
		out string
	}{
		{int64(1), "1"},
		{math.NaN(), "null"},
		{float64(1.23456), "1.2346"},
		{"1.2345", "\"1.2345\""},
		{time.Hour, "3600"},
		{"minpv", "\"minpv\""},
		{time.Time{}, "null"},
		{now, "\"" + now.Format(time.RFC3339) + "\""},
	}

	for _, tc := range tc {
		out, err := encode(tc.in)
		require.NoError(t, err)
		assert.Equal(t, tc.out, out)
	}
}

func TestEncodeSlice(t *testing.T) {
	tc := []struct {
		in  interface{}
		out string
	}{
		{[]string{"a", "b"}, `["a","b"]`},
		{[2]int64{1, 2}, `[1,2]`},
		{[]float64{1, math.NaN()}, `[1,null]`},
	}

	for _, tc := range tc {
		out, err := encodeSlice(tc.in)
		require.NoError(t, err)
		assert.Equal(t, tc.out, out)
	}
}
