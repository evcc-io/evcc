package metrics

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"gorm.io/gorm"
)

// Completed local days are rolled up once into meters_daily and meters_flows_daily,
// so month and year periods read a row per day instead of every 15min slot.

// energySum is the energy of an entity summed over a bucket of slots, and a
// row of meters_daily where the bucket is a local day
type energySum struct {
	Start              int64    `gorm:"column:day;primaryKey;autoIncrement:false"` // start of the bucket
	Meter              int      `gorm:"column:meter;primaryKey;autoIncrement:false"`
	Slots              int      `gorm:"-"` // slot path only, as SocTemp
	Energy             float64  `gorm:"column:energy"`
	ReturnEnergy       float64  `gorm:"column:return_energy"`
	SocTemp            *float64 `gorm:"-"`           // of a single slot
	Cost               *float64 `gorm:"column:cost"` // grid only, as in Slot
	PricedEnergy       *float64 `gorm:"column:priced_energy"`
	ReturnCost         *float64 `gorm:"column:return_cost"`
	PricedReturnEnergy *float64 `gorm:"column:priced_return_energy"`
}

func (energySum) TableName() string {
	return "meters_daily"
}

// flowDay is the flow attribution of a local day. A row exists for every
// rolled up day, the newest marks how far the rollup got.
type flowDay struct {
	Day  int64  `gorm:"column:day;primaryKey;autoIncrement:false"`
	Data string `gorm:"column:data"` // json flowAcc
}

func (flowDay) TableName() string {
	return "meters_flows_daily"
}

const (
	// a day is complete once its slots cannot change anymore: the last slot is
	// written when the next one starts, the rest is margin for late writes
	dailyDelay = 2 * time.Hour

	// days rolled up per transaction
	dailyChunk = 31
)

// dailyNow is replaced in tests
var dailyNow = time.Now

var (
	dailyBusy  sync.Mutex   // one rollup at a time
	dailyMu    sync.Mutex   // guards the rollup tables together with dailyEnd
	dailyEnd   atomic.Int64 // start of the first day not rolled up, 0 if none is
	dailyDirty atomic.Int64 // earliest change the rollup does not reflect yet
	dailyStale atomic.Int64 // earliest change since the running rollup read its slots
)

const dailyClean = math.MaxInt64

func dayStart(t time.Time) time.Time {
	return aggregateStarts["day"](t)
}

// atomicMin lowers v to ts
func atomicMin(v *atomic.Int64, ts int64) {
	for {
		if old := v.Load(); ts >= old || v.CompareAndSwap(old, ts) {
			return
		}
	}
}

// dailyFinal returns the end of the days that are complete
func dailyFinal() time.Time {
	return dayStart(dailyNow().Add(-dailyDelay))
}

// loadDaily restores how far the rollup got. A rollup of another time zone
// is dropped.
func loadDaily() error {
	dailyEnd.Store(0)
	dailyDirty.Store(dailyClean)

	var last flowDay
	tx := db.Instance.Select("day").Order("day DESC").Limit(1).Find(&last)
	if tx.Error != nil {
		return tx.Error
	}

	day := time.Unix(last.Day, 0)
	if tx.RowsAffected == 0 || !day.Equal(dayStart(day)) {
		return deleteDaily(db.Instance, 0, dailyClean)
	}

	dailyEnd.Store(day.AddDate(0, 0, 1).Unix())
	return nil
}

// deleteDaily removes the rolled up days in [from,to)
func deleteDaily(tx *gorm.DB, from, to int64) error {
	if err := tx.Where("day >= ? AND day < ?", from, to).Delete(new(energySum)).Error; err != nil {
		return err
	}
	return tx.Where("day >= ? AND day < ?", from, to).Delete(new(flowDay)).Error
}

// markDaily notes a change to the slots or tariffs at ts. It neither blocks
// nor fails, so it is safe on the write path. Readers stop using the rollup
// from that day on, the next rollup drops it.
func markDaily(ts int64) {
	if ts >= dailyFinal().Unix() {
		return // never rolled up
	}

	atomicMin(&dailyStale, ts)
	atomicMin(&dailyDirty, ts)
}

// invalidateDaily drops the rollup from the day of ts on
func invalidateDaily(ts int64) error {
	markDaily(ts)
	return applyDaily()
}

// applyDaily drops the rolled up days that markDaily noted a change for
func applyDaily() error {
	dailyMu.Lock()
	defer dailyMu.Unlock()

	dirty := dailyDirty.Load()
	if dirty == dailyClean {
		return nil
	}

	if dirty < dailyEnd.Load() {
		day := dayStart(time.Unix(dirty, 0)).Unix()
		if err := db.Instance.Transaction(func(tx *gorm.DB) error {
			return deleteDaily(tx, day, dailyClean)
		}); err != nil {
			return err
		}

		// nothing left: start over at the first slot, which may have moved
		var n int64
		if err := db.Instance.Model(new(flowDay)).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			day = 0
		}
		dailyEnd.Store(day)
	}

	// an earlier change that arrived meanwhile stays for the next run
	dailyDirty.CompareAndSwap(dirty, dailyClean)
	return nil
}

// dailyUsable returns the end of the rolled up days that reflect all changes
func dailyUsable() int64 {
	end := dailyEnd.Load()
	if dirty := dailyDirty.Load(); dirty < end {
		end = dayStart(time.Unix(dirty, 0)).Unix()
	}
	return end
}

// Rollup sums up all completed days that are not rolled up yet
func Rollup() error {
	return syncDaily(true)
}

// syncDaily rolls up completed days, all of them or the next chunk. It
// returns right away while another rollup is running.
func syncDaily(all bool) error {
	final := dailyFinal().Unix()
	if dailyDirty.Load() == dailyClean && dailyEnd.Load() >= final {
		return nil
	}

	if !dailyBusy.TryLock() {
		return nil
	}
	defer dailyBusy.Unlock()

	for {
		if err := applyDaily(); err != nil {
			return err
		}

		end := dailyEnd.Load()
		if end >= final {
			return nil
		}

		next, err := rollup(end, time.Unix(final, 0))
		if err != nil || next == end || !all {
			return err
		}
	}
}

// rollup stores the next chunk of days before final and returns its end. A
// chunk that a change arrived for meanwhile is discarded and returns -1.
func rollup(end int64, final time.Time) (int64, error) {
	dailyStale.Store(dailyClean)

	from := time.Unix(end, 0)
	if end == 0 {
		first, _, err := slotRange()
		if err != nil || first == 0 {
			return end, err
		}
		from = dayStart(time.Unix(first, 0))
	}

	to := from.AddDate(0, 0, dailyChunk)
	if to.After(final) {
		to = final
	}
	if !from.Before(to) {
		return end, nil
	}

	days := buckets(from, to, "day")

	sums, err := sumEnergy(days, false, nil)
	if err != nil {
		return end, err
	}

	flows := make([]flowDay, 0, len(days))
	for _, d := range days {
		acc, err := accumulate(time.Unix(d[0], 0), time.Unix(d[1], 0))
		if err != nil {
			return end, err
		}
		// most entities are idle on a given day
		maps.DeleteFunc(acc.Entities, func(_ int, e *flowEntity) bool {
			return e.Metered == 0 && e.Flows == [3]flowSum{}
		})
		data, err := json.Marshal(acc)
		if err != nil {
			return end, err
		}
		flows = append(flows, flowDay{Day: d[0], Data: string(data)})
	}

	dailyMu.Lock()
	defer dailyMu.Unlock()

	if dailyStale.Load() < to.Unix() {
		return -1, nil
	}

	// replaces what another process may have rolled up for these days
	if err := db.Instance.Transaction(func(tx *gorm.DB) error {
		if err := deleteDaily(tx, from.Unix(), to.Unix()); err != nil {
			return err
		}
		if len(sums) > 0 {
			if err := tx.CreateInBatches(sums, 1000).Error; err != nil {
				return err
			}
		}
		return tx.Create(flows).Error
	}); err != nil {
		return end, err
	}

	dailyEnd.Store(to.Unix())
	return to.Unix(), nil
}

// slotRange returns the first and last slot, 0 without slots. The lookup per
// entity is served by the index, unlike a plain MIN over all rows.
func slotRange() (int64, int64, error) {
	var res struct{ First, Last *int64 }
	if err := db.Instance.Raw(`SELECT MIN(first) AS first, MAX(last) AS last FROM (
		SELECT (SELECT MIN(ts) FROM meters WHERE meter = e.id) AS first,
			(SELECT MAX(ts) FROM meters WHERE meter = e.id) AS last
		FROM entities e)`).Scan(&res).Error; err != nil || res.First == nil {
		return 0, 0, err
	}
	return *res.First, *res.Last, nil
}

// closeBounds replaces zero bounds by the first and last of the given slots
func closeBounds(from, to time.Time, slots func() (int64, int64, error)) (time.Time, time.Time, error) {
	if !from.IsZero() && !to.IsZero() {
		return from, to, nil
	}

	first, last, err := slots()
	if from.IsZero() {
		from = time.Unix(first, 0)
	}
	if to.IsZero() {
		to = time.Unix(last, 0).Add(tariff.SlotDuration)
	}
	return from, to, err
}

// buckets splits [from,to) at the local boundaries of the aggregate
func buckets(from, to time.Time, aggregate string) [][2]int64 {
	start, next := aggregateStarts[aggregate], aggregateDurations[aggregate]

	var res [][2]int64
	for from.Before(to) {
		end := next(start(from))
		if end.After(to) {
			end = to
		}
		res = append(res, [2]int64{from.Unix(), end.Unix()})
		from = end
	}
	return res
}

// bucketValues lists the buckets as sql values
func bucketValues(buckets [][2]int64) string {
	var res strings.Builder
	for i, b := range buckets {
		if i > 0 {
			res.WriteByte(',')
		}
		fmt.Fprintf(&res, "(%d,%d)", b[0], b[1])
	}
	return res.String()
}

// wholeDays returns the rolled up days [d0,d1) inside [from,to). Zero bounds are open ended.
func wholeDays(from, to time.Time, end int64) (int64, int64) {
	var d0 int64
	if !from.IsZero() {
		day := dayStart(from)
		if day.Before(from) {
			day = day.AddDate(0, 0, 1)
		}
		d0 = day.Unix()
	}

	d1 := end
	if !to.IsZero() {
		d1 = min(d1, dayStart(to).Unix())
	}

	return d0, d1
}

const energySumColumns = `COUNT(*), COALESCE(SUM(m.energy), 0), COALESCE(SUM(m.return_energy), 0),
	CASE WHEN COUNT(*) = 1 THEN m.soc_temp END,
	SUM(m.energy * t.grid), SUM(CASE WHEN t.grid IS NOT NULL THEN m.energy END),
	SUM(m.return_energy * t.feedin), SUM(CASE WHEN t.feedin IS NOT NULL THEN m.return_energy END)`

// sumEnergy sums the slots per entity and bucket, or per slot of the buckets.
// The entities lead the join so only the slots of the buckets are read, the
// tariff is looked up for the grid only. A nil ids selects all entities.
func sumEnergy(buckets [][2]int64, perSlot bool, ids []int) ([]energySum, error) {
	if len(buckets) == 0 {
		return nil, nil
	}

	start := "b.s"
	if perSlot {
		start = "m.ts"
	}

	query := `WITH b(s, e) AS (VALUES ` + bucketValues(buckets) + `)
		SELECT ` + start + `, m.meter, ` + energySumColumns + `
		FROM entities e
		CROSS JOIN b
		CROSS JOIN meters m ON m.meter = e.id AND m.ts >= b.s AND m.ts < b.e
		LEFT JOIN tariffs t ON t.ts = m.ts AND e."group" = 'grid'`

	var args []any
	if ids != nil {
		query += ` WHERE e.id IN ?`
		args = append(args, ids)
	}

	return scanEnergy(db.Instance.Raw(query+` GROUP BY e.id, `+start, args...))
}

// scanEnergy scans by hand, gorm's reflective scan dominates long periods
func scanEnergy(tx *gorm.DB) ([]energySum, error) {
	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []energySum
	for rows.Next() {
		var s energySum
		if err := rows.Scan(&s.Start, &s.Meter, &s.Slots, &s.Energy, &s.ReturnEnergy, &s.SocTemp,
			&s.Cost, &s.PricedEnergy, &s.ReturnCost, &s.PricedReturnEnergy); err != nil {
			return nil, err
		}
		res = append(res, s)
	}

	return res, rows.Err()
}

// dailyEnergy sums [from,to) per entity and local day. Rolled up days are
// read from the rollup, the rest from the slots.
func dailyEnergy(from, to time.Time, ids []int) ([]energySum, error) {
	// a failed rollup leaves the days to the slots
	_ = syncDaily(false)

	d0, d1 := wholeDays(from, to, dailyUsable())

	var res []energySum
	if d0 < d1 {
		tx := db.Instance.Model(new(energySum)).
			Select("day, meter, 0, energy, return_energy, NULL, cost, priced_energy, return_cost, priced_return_energy").
			Where("day >= ? AND day < ?", d0, d1)
		if ids != nil {
			tx = tx.Where("meter IN ?", ids)
		}

		var err error
		if res, err = scanEnergy(tx); err != nil {
			return nil, err
		}
	}

	var rest [][2]int64
	for _, d := range buckets(from, to, "day") {
		if d[0] < d0 || d[0] >= d1 {
			rest = append(rest, d)
		}
	}

	slots, err := sumEnergy(rest, false, ids)
	return append(res, slots...), err
}

// dailyFlow attributes [from,to). Rolled up days are read from the rollup,
// the rest is attributed from the slots. Zero bounds are open ended.
func dailyFlow(from, to time.Time) (*flowAcc, error) {
	// a failed rollup leaves the days to the slots
	_ = syncDaily(false)

	d0, d1 := wholeDays(from, to, dailyUsable())
	if d0 >= d1 {
		return accumulate(from, to)
	}

	var days []flowDay
	if err := db.Instance.Select("data").Where("day >= ? AND day < ?", d0, d1).Find(&days).Error; err != nil {
		return nil, err
	}

	acc := &flowAcc{Entities: make(map[int]*flowEntity)}
	for _, d := range days {
		var day flowAcc
		if err := json.Unmarshal([]byte(d.Data), &day); err != nil {
			return nil, err
		}
		acc.merge(&day)
	}

	// the slots before and after the rolled up days
	if !from.IsZero() && from.Unix() < d0 {
		head, err := accumulate(from, time.Unix(d0, 0))
		if err != nil {
			return nil, err
		}
		acc.merge(head)
	}
	if to.IsZero() || d1 < to.Unix() {
		tail, err := accumulate(time.Unix(d1, 0), to)
		if err != nil {
			return nil, err
		}
		acc.merge(tail)
	}

	return acc, nil
}
