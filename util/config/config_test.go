package config

import (
	"testing"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
)

func TestPreferencesRoundTrip(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(new(Config)))

	// row without preferences, as written before the column existed
	require.NoError(t, db.Instance.Exec(`INSERT INTO configs (id, class, type, value) VALUES (1, ?, 'template', '{"a":1}')`, templates.Meter).Error)

	conf, err := ConfigByID(1)
	require.NoError(t, err)
	require.Equal(t, Preferences{}, conf.Preferences)

	require.NoError(t, conf.Update(conf.Data, WithProperties(Properties{Type: "template", Preferences: Preferences{DemandPredictor: "weekday"}})))

	conf, err = ConfigByID(1)
	require.NoError(t, err)
	require.Equal(t, "weekday", conf.Preferences.DemandPredictor)
}
