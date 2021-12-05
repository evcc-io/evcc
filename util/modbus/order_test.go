package modbus

import (
	"bytes"
	"testing"
)

func TestOrdered(t *testing.T) {
	if b := Ordered("", nil); b != nil {
		t.Errorf("expected nil, got %0x", b)
	}

	if b := Ordered("A", nil); b != nil {
		t.Errorf("expected nil, got %0x", b)
	}

	if b := Ordered("A", []byte{0, 0}); b != nil {
		t.Errorf("expected nil, got %0x", b)
	}

	res := []byte{0, 1, 2, 3}
	if b := Ordered("ABCD", []byte{0, 1, 2, 3}); !bytes.Equal(b, res) {
		t.Errorf("expected %0x, got %0x", res, b)
	}

	res = []byte{3, 2, 1, 0}
	if b := Ordered("DCBA", []byte{0, 1, 2, 3}); !bytes.Equal(b, res) {
		t.Errorf("expected %0x, got %0x", res, b)
	}
}

func TestOrderedM(t *testing.T) {
	if u := Ordered16("AB", []byte{1, 2}); u != 0x0102 {
		t.Errorf("expected 0x0102, got %0x", u)
	}

	if u := Ordered16("BA", []byte{1, 2}); u != 0x0201 {
		t.Errorf("expected 0x0201, got %0x", u)
	}

	if u := Ordered32("DCBA", []byte{4, 3, 2, 1}); u != 0x01020304 {
		t.Errorf("expected 0x01020304, got %0x", u)
	}

	if u := Ordered64("HGFEDCBA", []byte{8, 7, 6, 5, 4, 3, 2, 1}); u != 0x0102030405060708 {
		t.Errorf("expected 0x0102030405060708, got %0x", u)
	}
}
