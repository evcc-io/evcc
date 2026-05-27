package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
)

func TestMigrateLegacyHemsSetting(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(&config.Config{}))

	const legacyYaml = `type: relay
maxpower: 4200
limit:
  source: mqtt
  topic: hems/limit/status
`
	settings.SetString(keys.Hems, legacyYaml)

	require.NoError(t, migrateLegacyHemsSetting())

	require.False(t, settings.Exists(keys.Hems))

	devices, err := config.ConfigurationsByClass(templates.Hems)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "custom", devices[0].Type)
	require.Equal(t, legacyYaml, devices[0].Data["yaml"])

	// second run is a noop
	require.NoError(t, migrateLegacyHemsSetting())
	devices, err = config.ConfigurationsByClass(templates.Hems)
	require.NoError(t, err)
	require.Len(t, devices, 1)
}

func TestMigrateLegacyHemsSetting_noop(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(&config.Config{}))

	require.NoError(t, migrateLegacyHemsSetting())

	devices, err := config.ConfigurationsByClass(templates.Hems)
	require.NoError(t, err)
	require.Empty(t, devices)
}
