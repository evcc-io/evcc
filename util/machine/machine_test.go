package machine

import (
	"errors"
	"testing"

	"github.com/denisbrodbeck/machineid"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
)

func TestProtectedMachineId(t *testing.T) {
	const key = "foo"

	if mid, err := machineid.ProtectedID(key); err == nil {
		id := ProtectedID(key)

		if mid != id {
			t.Errorf("machine id mismatch. expected %s, got %s", mid, id)
		}
	} else {
		t.Skip("cannot get machineid, skipping test")
	}
}

func TestIdFromSettings(t *testing.T) {
	// reset machine id cache
	id = ""

	// reset settings
	settings.Delete(keys.Plant)

	// force machineid.ID() to fail
	getMachineID = func() (string, error) {
		return "", errors.New("dummy error")
	}
	t.Cleanup(func() {
		getMachineID = machineid.ID
	})

	// generate new random id
	generatedId := ID()
	if len(generatedId) != 64 {
		t.Errorf("expected 64 char id, got %d", len(generatedId))
	}

	// check that id is stored in settings
	settingsId, _ := settings.String(keys.Plant)
	if generatedId != settingsId {
		t.Errorf("expected id %s, got %s from settings", generatedId, settingsId)
	}

	// check reproducability
	idA1 := ProtectedID("A")
	idA2 := ProtectedID("A")
	idB := ProtectedID("B")

	if idA1 != idA2 {
		t.Errorf("expected same id, got %s and %s", idA1, idA2)
	}

	if idA1 == idB {
		t.Errorf("expected different id, got %s and %s", idA1, idB)
	}
}
