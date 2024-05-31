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
	expect := "http://localhost/a={{b}}?a={{b}}"

	if uri := DefaultScheme("localhost/a={{b}}?a={{b}}", "http"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}

	if uri := DefaultScheme("http://localhost/a={{b}}?a={{b}}", "http"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}

	if uri := DefaultScheme("http://localhost/a={{b}}?a={{b}}", "https"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}

	expect = "ws://localhost:8080/a={{b}}?a={{b}}"

	if uri := DefaultScheme("localhost:8080/a={{b}}?a={{b}}", "ws"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}
}

func TestDefaultSchemeWithEmptyUri(t *testing.T) {
	expect := ""

	if uri := DefaultScheme("", "http"); uri != expect {
		t.Errorf("expected %s, got %s", expect, uri)
	}
}
