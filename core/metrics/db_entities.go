package metrics

import (
	"database/sql"
	"time"

	"github.com/evcc-io/evcc/server/db"
)

// EntityInfo describes a metric entity and the extent of its stored data.
type EntityInfo struct {
	Group string
	Name  string
	Slots int       // number of persisted 15min slots
	First time.Time // start of the earliest slot, zero if no data
	Last  time.Time // start of the latest slot, zero if no data
}

// ListEntities returns all metric entities together with their slot count and
// data range. Entities without any persisted slots are included with a zero
// slot count and zero First/Last timestamps.
func ListEntities() ([]EntityInfo, error) {
	type row struct {
		Group string
		Name  string
		Slots int
		First sql.NullInt64
		Last  sql.NullInt64
	}

	var rows []row
	if err := db.Instance.Table("entities e").
		Select(`e."group" AS "group", e.name AS name,
			COUNT(m.ts) AS slots,
			MIN(m.ts) AS first, MAX(m.ts) AS last`).
		Joins("LEFT JOIN meters m ON m.meter = e.id").
		Group("e.id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	res := make([]EntityInfo, 0, len(rows))
	for _, r := range rows {
		e := EntityInfo{Group: r.Group, Name: r.Name, Slots: r.Slots}
		if r.First.Valid {
			e.First = time.Unix(r.First.Int64, 0)
		}
		if r.Last.Valid {
			e.Last = time.Unix(r.Last.Int64, 0)
		}
		res = append(res, e)
	}

	return res, nil
}
