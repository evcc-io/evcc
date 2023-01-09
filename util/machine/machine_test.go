package machine

import (
	"testing"

	"github.com/denisbrodbeck/machineid"
)

func TestProtectedMachineId(t *testing.T) {
	const key = "foo"

	if mid, err := machineid.ProtectedID(key); err == nil {
		id, err := ProtectedID(key)
		if err != nil {
			t.Error(err)
		}

		if mid != id {
			t.Errorf("machine id mismatch. expected %s, got %s", mid, id)
		}
	}
}
