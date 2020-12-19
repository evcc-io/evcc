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

func TestDefaultScheme(t *testing.T) {
	expect := "http://localhost"

	if uri := DefaultScheme("localhost", "http"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}

	if uri := DefaultScheme("http://localhost", "http"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}

	if uri := DefaultScheme("http://localhost", "https"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}

	expect = "ws://localhost:8080"

	if uri := DefaultScheme("localhost:8080", "ws"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}
}
