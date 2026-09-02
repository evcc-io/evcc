package settings

import (
	"testing"

	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistClearsDirty guards against clearing the flag on a range copy, which
// would re-save every setting ever changed on each subsequent Persist.
func TestPersistClearsDirty(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	t.Cleanup(func() { db.Instance = nil })

	SetString("persisted", "foo")
	require.NoError(t, Persist())

	mu.RLock()
	defer mu.RUnlock()

	for _, s := range settings {
		assert.False(t, s.dirty, "setting %s still dirty after persist", s.Key)
	}
}
