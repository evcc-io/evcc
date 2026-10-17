package shm

import (
	"bytes"
	"testing"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/google/uuid"
)

func TestPersistentUUID(t *testing.T) {
	settings.Delete(keys.ShmUuid)
	t.Cleanup(func() { settings.Delete(keys.ShmUuid) })

	uid := persistentUUID()
	if !bytes.Equal(uid[10:], uuid.NodeID()) {
		t.Errorf("expected node %x, got %x", uuid.NodeID(), uid[10:])
	}

	if s, _ := settings.String(keys.ShmUuid); s != uid.String() {
		t.Errorf("expected setting %s, got %s", uid, s)
	}

	if again := persistentUUID(); again != uid {
		t.Errorf("expected persisted %s, got %s", uid, again)
	}
}
