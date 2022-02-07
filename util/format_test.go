package util

import (
	"math"
	"testing"
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
		b := Truish(c.k)
		if b != c.v {
			t.Errorf("expected %v got %v", c.v, b)
		}
	}
}

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
		s, err := ReplaceFormatted(c.fmt, map[string]interface{}{
			c.k: c.v,
		})

		if s != c.expected || err != nil {
			t.Error(s, err)
		}
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

func TestTemplateReplaceFormatted(t *testing.T) {
	msg := "Wallbox {{.title}} started charging {{.vehicleTitle}} in {{.mode}} mode"
	s, err := ReplaceFormatted(msg, map[string]interface{}{
		"title":        "go-e",
		"vehicleTitle": "Zoe",
		"mode":         "pv",
	})

	if err != nil {
		t.Error(s, err)
	}
}
