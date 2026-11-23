package metrics

import (
	"testing"

	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/require"
)

func meterTableDDL(t *testing.T) string {
	t.Helper()
	var ddl string
	require.NoError(t, db.Instance.Raw(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'meters'`).Scan(&ddl).Error)
	return ddl
}

func TestMeterWithoutRowid(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())
	require.Contains(t, meterTableDDL(t), withoutRowid)

	// a second start keeps the table as it is
	require.NoError(t, SetupSchema())
	require.Contains(t, meterTableDDL(t), withoutRowid)
}

func TestMeterWithoutRowidMigration(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	// replace meters by the schema of a database from before, created by AutoMigrate with the unique index
	for _, stmt := range []string{
		"DROP TABLE meters",
		"INSERT INTO entities (id, `group`, name, title) VALUES (2, 'grid', 'db:1', 'grid')",
		"CREATE TABLE \"meters\"  (`meter` integer,`ts` integer,\"energy\" real,\"return_energy\" real, `soc_temp` real, `recovered` numeric,CONSTRAINT `fk_meters_entity` FOREIGN KEY (`meter`) REFERENCES `entities`(`id`))",
		"CREATE UNIQUE INDEX `meters_meter_ts` ON `meters`(`meter`,`ts`)",
		"INSERT INTO meters VALUES (2, 1800, 0.5, 0.1, NULL, 0), (2, 900, 0.25, 0, NULL, 1), (1, 900, 0.3, 0, 21.5, 0)",
	} {
		require.NoError(t, db.Instance.Exec(stmt).Error, stmt)
	}

	require.NoError(t, SetupSchema())
	require.Contains(t, meterTableDDL(t), withoutRowid)
	require.False(t, db.Instance.Migrator().HasTable("meters_rowid"))
	require.False(t, db.Instance.Migrator().HasIndex(new(meter), "meters_meter_ts"))

	var rows []meter
	require.NoError(t, db.Instance.Order("meter, ts").Find(&rows).Error)
	require.Len(t, rows, 3)
	require.Equal(t, 1, rows[0].Meter)
	require.InDelta(t, 21.5, *rows[0].SocTemp, 1e-9)
	require.Equal(t, meter{Meter: 2, Timestamp: 900, Energy: 0.25, Recovered: 1}, rows[1])
	require.Equal(t, meter{Meter: 2, Timestamp: 1800, Energy: 0.5, ReturnEnergy: 0.1}, rows[2])

	// the primary key still rejects a second row for the same slot
	require.Error(t, db.Instance.Create(&meter{Meter: 2, Timestamp: 900}).Error)
}
