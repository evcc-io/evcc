package provider

import (
	"math"
	"testing"
)

func TestReplace(t *testing.T) {
	cases := []struct {
		k             string
		v             interface{}
		fmt, expected string
	}{
		{"foo", true, "${foo}", "true"},
		{"foo", "1", "abc${foo}${foo}", "abc11"},
		{"foo", math.Pi, "${foo:%.2f}", "3.14"},
	}

	for _, c := range cases {
		s, err := replaceFormatted(c.fmt, map[string]interface{}{
			c.k: c.v,
		})

		if s != c.expected || err != nil {
			t.Error(s, err)
		}
	}
}

func TestReplaceMulti(t *testing.T) {
	s, err := replaceFormatted("${foo}-${bar}", map[string]interface{}{
		"foo": "bar",
		"bar": "baz",
	})

	if s != "bar-baz" || err != nil {
		t.Error(s, err)
	}
}

func TestReplaceNoMatch(t *testing.T) {
	s, err := replaceFormatted("${foo}", map[string]interface{}{
		"bar": "baz",
	})

	if s != "" || err == nil {
		t.Error(s, err)
	}
}
