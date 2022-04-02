package util

import (
	"testing"
)

func TestTore(t *testing.T) {

	s := NewStore("evcc_test", "")

	s.Open()

	s.Put("key42", "this is a string")

	s.Close()

}
