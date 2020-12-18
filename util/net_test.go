package util

import (
	"testing"
)

// DefaultPort appends given port to connection if not specified
func TestDefaultPort(t *testing.T) {
	expect := "foo:7090"

	if uri := DefaultPort("foo:7090", 7090); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}

	if uri := DefaultPort("foo", 7090); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}
}
