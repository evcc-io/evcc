package pipeline

import (
	"bytes"
	"testing"
)

func TestRegex(t *testing.T) {
	for _, re := range []string{`([0-9.]+)`, `[0-9.]+`} {
		p, err := new(Pipeline).WithRegex(re)
		if err != nil {
			t.Error(err)
		}

		res, err := p.Process([]byte("12.3W"))
		if err != nil {
			t.Error(err)
		}

		if exp := []byte("12.3"); !bytes.Equal(res, exp) {
			t.Errorf("Expected %s, got %s", exp, res)
		}
	}
}
