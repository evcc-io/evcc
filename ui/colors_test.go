package ui

import (
	"path/filepath"
	"testing"

	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDB(t *testing.T) {
	t.Helper()
	require.NoError(t, db.NewInstance("sqlite", filepath.Join(t.TempDir(), "test.db")))
}

func TestDeviceColors_RoundTrip(t *testing.T) {
	setupDB(t)

	in := map[string]string{"WP-SG+": "#2563EB", "Heizung": "#DC2626"}
	require.NoError(t, SaveDeviceColors(in))

	assert.Equal(t, in, GetDeviceColors())
}

func TestDeviceColors_EmptyWhenAbsent(t *testing.T) {
	setupDB(t)
	assert.Empty(t, GetDeviceColors())
}

func TestDeviceColorList_SortedAndSafe(t *testing.T) {
	setupDB(t)

	require.NoError(t, SaveDeviceColors(map[string]string{
		"WP-SG+":  "#2563EB",
		"Heizung": "#DC2626",
		"Carport": "#10B981",
	}))

	assert.Equal(t, []DeviceColor{
		{Title: "Carport", Color: "#10B981"},
		{Title: "Heizung", Color: "#DC2626"},
		{Title: "WP-SG+", Color: "#2563EB"},
	}, DeviceColorList())
}
