package util

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruish(t *testing.T) {
	cases := []struct {
		k string
		v bool
	}{
		{"", false},
		{"false", false},
		{"0", false},
		{"off", false},
		{"true", true},
		{"1", true},
		{"on", true},
	}

	for _, c := range cases {
		assert.Equal(t, c.v, Truish(c.k))
	}
}

func TestReplace(t *testing.T) {
	cases := []struct {
		k             string
		v             interface{}
		fmt, expected string
	}{
		// regex tests
		{"foo", true, "${foo}", "true"},
		{"foo", true, "${Foo}", "true"},
		{"Foo", true, "${foo}", "true"},
		{"foo", "1", "abc${foo}${foo}", "abc11"},
		{"foo", math.Pi, "${foo:%.2f}", "3.14"},
		{"foo", math.Pi, "${foo:%.0f}%", "3%"},
		{"foo", 3, "${foo}%", "3%"},
	}

	for _, c := range cases {
		s, err := ReplaceFormatted(c.fmt, map[string]interface{}{
			c.k: c.v,
		})

		require.NoError(t, err)
		assert.Equal(t, c.expected, s)
	}
}

func TestReplaceMulti(t *testing.T) {
	s, err := ReplaceFormatted("${foo}-${bar}", map[string]interface{}{
		"foo": "bar",
		"bar": "baz",
	})

	if s != "bar-baz" || err != nil {
		t.Error(s, err)
	}
}

func TestReplaceNoMatch(t *testing.T) {
	s, err := ReplaceFormatted("${foo}", map[string]interface{}{
		"bar": "baz",
	})

	if err == nil {
		t.Error(s, err)
	}
}

func TestReplaceTemplate(t *testing.T) {
	tc := []struct {
		in, out, key string
		val          any
	}{
		{`"{{ .mode }}"`, `"pv"`, "mode", "pv"},
		{`{{ printf "%.1f" .chargedEnergy }}kW`, `1.2kW`, "chargedEnergy", 1.234},
		{`{{ round .chargedEnergy 1 }}kW`, `1.2kW`, "chargedEnergy", 1.234},
		{`{{ timeRound .connectedDuration "s" }}`, `1s`, "connectedDuration", 1234 * time.Millisecond},
		{`{{ timeRound .connectedDuration "m" }}`, `21m0s`, "connectedDuration", 1234 * time.Second},
	}

	for _, tc := range tc {
		s, err := ReplaceFormatted(tc.in, map[string]interface{}{
			tc.key: tc.val,
		})

		t.Log(s)
		if err != nil {
			t.Error(s, err)
		}

		if s != tc.out {
			t.Errorf("expected: %s, got: %s", tc.out, s)
		}
	}
}
