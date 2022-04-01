package util

import (
	"testing"
)

func TestTore(t *testing.T) {

	s, err := NewStore("evcc_test")
	if err != nil {
		t.Errorf("OpenStore %v", err)
	}

	t.Logf("Db %v", s.db)

	err = s.Put("key42", "this is a string")
	if err != nil {
		t.Errorf("PutStore %v", err)
	}

	t.Logf("Store %v", s)

	err = s.CloseStore()
	if err != nil {
		t.Errorf("Close %v", err)
	}

}
