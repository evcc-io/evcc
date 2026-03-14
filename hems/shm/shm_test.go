package shm

import (
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchUUIDNode(t *testing.T) {
	original, err := uuid.Parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	require.NoError(t, err)

	want, err := hex.DecodeString("2e50ba116910")
	require.NoError(t, err)

	patched := patchUUIDNode(original, want)

	assert.Equal(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8", original.String())
	assert.Equal(t, "6ba7b810-9dad-11d1-80b4-2e50ba116910", patched.String())
}
