package metrics

import (
	"slices"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestSqliteTimestamp(t *testing.T) {
	clock := clock.NewMock()
	clock.Add(time.Hour)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	entity := entity{Name: "foo"}
	require.NoError(t, db.Instance.FirstOrCreate(&entity).Error)

	persist(entity, clock.Now(), 0, 0)

	db, err := db.Instance.DB()
	require.NoError(t, err)

	var ts SqlTime

	for _, sql := range []string{
		`SELECT ts FROM meters`,
		`SELECT min(ts) FROM meters`,
		// `SELECT unixepoch(ts) FROM meters`,
		// `SELECT unixepoch(min(ts)) FROM meters`,
		`SELECT min(ts) AS ts FROM meters
			GROUP BY strftime("%H:%M", ts)
			ORDER BY ts`,
	} {
		require.NoError(t, db.QueryRow(sql).Scan(&ts))
		require.True(t, clock.Now().Equal(time.Time(ts)), "expected %v, got %v (%s)", clock.Now().Local(), time.Time(ts).Local(), sql)
	}

	require.NoError(t, db.QueryRow(`SELECT ts FROM meters WHERE ts >= ?`, clock.Now().Unix()).Scan(&ts))
	require.True(t, clock.Now().Equal(time.Time(ts)), "expected %v, got %v", clock.Now().Local(), time.Time(ts).Local())
}

func TestQueryImportEnergyUTCFilter(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	e := entity{Name: Grid, Group: Grid}
	require.NoError(t, db.Instance.FirstOrCreate(&e).Error)

	// 2 hourly slots at 16:00 and 17:00 local time
	loc := time.Now().Location()
	base := time.Date(2026, 4, 15, 16, 0, 0, 0, loc)

	require.NoError(t, persist(e, base, 0, 1))
	require.NoError(t, persist(e, base.Add(time.Hour), 0, 2))

	// query with UTC times spanning both slots
	from := base.Add(-time.Hour).UTC()
	to := base.Add(3 * time.Hour).UTC()

	res, err := QueryImportEnergy(from, to, "hour", false)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[0].Data, 2)
	require.InDelta(t, 1, res[0].Data[0].Export, 0.001)
	require.InDelta(t, 2, res[0].Data[1].Export, 0.001)
}

func TestQueryImportEnergyGrouped(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	// two entities sharing the same group, different names
	e1 := entity{Id: 2, Name: "db:12", Group: Grid}
	require.NoError(t, db.Instance.Create(&e1).Error)
	e2 := entity{Id: 3, Name: "db:13", Group: Grid}
	require.NoError(t, db.Instance.Create(&e2).Error)

	loc := time.Now().Location()
	base := time.Date(2026, 4, 15, 16, 0, 0, 0, loc)

	require.NoError(t, persist(e1, base, 1, 0))
	require.NoError(t, persist(e2, base, 2, 0))
	require.NoError(t, persist(e1, base.Add(time.Hour), 3, 0))
	require.NoError(t, persist(e2, base.Add(time.Hour), 4, 0))

	from := base.Add(-time.Hour).UTC()
	to := base.Add(3 * time.Hour).UTC()

	// ungrouped: 2 series
	res, err := QueryImportEnergy(from, to, "hour", false)
	require.NoError(t, err)
	require.Len(t, res, 2)

	// grouped: 1 series, values summed per bucket
	res, err = QueryImportEnergy(from, to, "hour", true)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, Grid, res[0].Group)
	require.Empty(t, res[0].Name)
	require.Len(t, res[0].Data, 2)
	require.InDelta(t, 1+2, res[0].Data[0].Import, 0.001)
	require.InDelta(t, 3+4, res[0].Data[1].Import, 0.001)
}

func TestQueryImportEnergyMultipleSeries(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	// 3 entities across 2 groups
	eGrid := entity{Id: 2, Name: Grid, Group: Grid}
	require.NoError(t, db.Instance.Create(&eGrid).Error)
	ePv1 := entity{Id: 4, Name: "pv1", Group: PV}
	require.NoError(t, db.Instance.Create(&ePv1).Error)
	ePv2 := entity{Id: 5, Name: "pv2", Group: PV}
	require.NoError(t, db.Instance.Create(&ePv2).Error)

	loc := time.Now().Location()
	base := time.Date(2026, 4, 15, 16, 0, 0, 0, loc)

	// 2 hourly slots per entity
	for i := range 2 {
		ts := base.Add(time.Duration(i) * time.Hour)
		require.NoError(t, persist(eGrid, ts, float64(1+i), 0))
		require.NoError(t, persist(ePv1, ts, 0, float64(10+i)))
		require.NoError(t, persist(ePv2, ts, 0, float64(20+i)))
	}

	from := base.Add(-time.Hour).UTC()
	to := base.Add(3 * time.Hour).UTC()

	// ungrouped: 3 series, each with 2 slots
	res, err := QueryImportEnergy(from, to, "hour", false)
	require.NoError(t, err)
	require.Len(t, res, 3)

	byName := map[string]Series{}
	for _, s := range res {
		require.Len(t, s.Data, 2)
		byName[s.Name] = s
	}
	require.Equal(t, Grid, byName[Grid].Group)
	require.Equal(t, PV, byName["pv1"].Group)
	require.Equal(t, PV, byName["pv2"].Group)

	require.InDelta(t, 1, byName[Grid].Data[0].Import, 0.001)
	require.InDelta(t, 2, byName[Grid].Data[1].Import, 0.001)
	require.InDelta(t, 10, byName["pv1"].Data[0].Export, 0.001)
	require.InDelta(t, 21, byName["pv2"].Data[1].Export, 0.001)

	// grouped: 2 series, pv summed per bucket
	res, err = QueryImportEnergy(from, to, "hour", true)
	require.NoError(t, err)
	require.Len(t, res, 2)

	byGroup := map[string]Series{}
	for _, s := range res {
		require.Empty(t, s.Name)
		require.Len(t, s.Data, 2)
		byGroup[s.Group] = s
	}
	require.Contains(t, byGroup, Grid)
	require.Contains(t, byGroup, PV)

	require.InDelta(t, 1, byGroup[Grid].Data[0].Import, 0.001)
	require.InDelta(t, 2, byGroup[Grid].Data[1].Import, 0.001)
	require.InDelta(t, 10+20, byGroup[PV].Data[0].Export, 0.001)
	require.InDelta(t, 11+21, byGroup[PV].Data[1].Export, 0.001)
}

func TestUpdateProfile(t *testing.T) {
	clock := clock.NewMock()

	// adjust for 00:00 in local timezone
	_, o := clock.Now().Zone()
	clock.Add(-time.Duration(o) * time.Second)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	entity := entity{Id: 2, Name: "foo"}
	require.NoError(t, db.Instance.FirstOrCreate(&entity).Error)

	// 2 days of data
	// day 1:   0 ...  95
	// day 2:  96 ... 181
	for i := range 4 * 2 * 24 {
		persist(entity, clock.Now(), float64(i), float64(i))
		clock.Add(15 * time.Minute)
	}

	// validate records written
	var count int64
	require.NoError(t, db.Instance.Model(new(meter)).Count(&count).Error)
	require.Equal(t, int64(24*2*4), count)

	{
		from := clock.Now().Local().AddDate(0, 0, -2).Add(12 * time.Hour) // 12:00 of day 0

		prof, err := importProfile(entity, from)
		require.NoError(t, err)

		var expected [96]float64
		for i := range expected {
			if i < 48 {
				expected[i] = float64(48+i+144+i) / 2
				continue
			}
			expected[i] = float64(96 - 48 + i)
		}

		require.Equal(t, expected, *prof, "partial profile: expected %v, got %v", expected, *prof)
	}

	{
		from := clock.Now().Local().AddDate(0, 0, -3).Add(12 * time.Hour) // 12:00 of day -1

		prof, err := importProfile(entity, from)
		require.NoError(t, err)

		var expected [96]float64
		for i := range expected {
			expected[i] = float64(0+96+2*i) / 2
		}

		require.Equal(t, expected, *prof, "full profile: expected %v, got %v", expected, *prof)
	}
}

func TestTimeMigration(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	mig := db.Instance.Migrator()

	require.NoError(t, db.Instance.AutoMigrate(new(entity)))

	type v1 struct {
		Meter     int       `json:"meter" gorm:"column:meter;uniqueIndex:idx_meter_ts"`
		Timestamp time.Time `json:"ts" gorm:"column:ts;uniqueIndex:idx_meter_ts"`
		Entity    entity    `json:"-" gorm:"foreignkey:Meter;references:Id"`
	}

	require.NoError(t, db.Instance.AutoMigrate(new(v1)))
	{
		tables, err := mig.GetTables()
		require.NoError(t, err)
		require.True(t, slices.Contains(tables, "v1"))
	}

	require.NoError(t, mig.RenameTable("v1", "v2"))

	type v2 struct {
		Meter     int    `json:"meter" gorm:"column:meter;uniqueIndex:idx_meter_ts"`
		Timestamp int64  `json:"ts" gorm:"column:ts;uniqueIndex:idx_meter_ts"`
		Entity    entity `json:"-" gorm:"foreignkey:Meter;references:Id"`
	}

	require.NoError(t, db.Instance.AutoMigrate(new(v2)))
	{
		tables, err := mig.GetTables()
		require.NoError(t, err)
		require.False(t, slices.Contains(tables, "v1"))
		require.True(t, slices.Contains(tables, "v2"))
	}
}
