package metrics

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
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

// flowDay holds the totals of a local day per source and sink (pv to home, ...)
// with cost and co2. A day is rolled up if and only if it has a row here.
type flowDay struct {
	Day  int64  `gorm:"column:day;primaryKey;autoIncrement:false"`
	Data string `gorm:"column:data"` // json flowAcc
}

func (flowDay) TableName() string {
	return "meters_flows_daily"
}

const (
	// a day is complete once every entity wrote its last slot, which is up to
	// two slots after midnight (advanceSlot, fill). The rest is margin.
	dailyDelay = 2 * time.Hour

	// days rolled up per transaction
	dailyChunk = 31
)

var log = util.NewLogger("metrics")

var (
	// held while days are rolled up and while completed days change, so no
	// day is rolled up from slots that change meanwhile
	dailyMu      sync.Mutex
	dailyRequest = make(chan struct{}, 1) // wakes up Rollup
)

func dayStart(t time.Time) time.Time {
	return aggregateStarts["day"](t)
}

// nextDay returns the start of the day after the one starting at day.
// AddDate misses it where daylight saving skips midnight.
func nextDay(day time.Time) time.Time {
	return aggregateDurations["day"](day)
}

// wholeDay reports whether the bucket is a local day from start to end
func wholeDay(b [2]int64) bool {
	day := time.Unix(b[0], 0)
	return day.Equal(dayStart(day)) && nextDay(day).Unix() == b[1]
}

// dailyFinal returns the end of the days that are complete at now
func dailyFinal(now time.Time) time.Time {
	return dayStart(now.Add(-dailyDelay))
}

// loadDaily drops a rollup of another time zone. A change to what a day sums
// up to needs the same: add a version to flowDay and drop on a mismatch.
func loadDaily() error {
	var last flowDay
	tx := db.Instance.Select("day").Order("day DESC").Limit(1).Find(&last)
	if tx.Error != nil {
		return tx.Error
	}

	if day := time.Unix(last.Day, 0); tx.RowsAffected > 0 && !day.Equal(dayStart(day)) {
		return resetDaily(nil)
	}
	return nil
}

// deleteDaily removes the rolled up days in [from,to). Both tables change in
// the transaction tx, a day with only one of them is read wrong.
func deleteDaily(tx *gorm.DB, from, to int64) error {
	if err := tx.Where("day >= ? AND day < ?", from, to).Delete(new(energySum)).Error; err != nil {
		return err
	}
	return tx.Where("day >= ? AND day < ?", from, to).Delete(new(flowDay)).Error
}

// resetDaily drops the whole rollup. A given change runs along with it and
// reports whether it applied, otherwise the rollup stays.
func resetDaily(change func(tx *gorm.DB) bool) error {
	dailyMu.Lock()
	defer dailyMu.Unlock()

	if err := db.Instance.Transaction(func(tx *gorm.DB) error {
		if change != nil && !change(tx) {
			return nil
		}
		return deleteDaily(tx, math.MinInt64, math.MaxInt64)
	}); err != nil {
		return err
	}

	requestRollup()
	return nil
}

// ResetRollup drops the whole rollup, for slots that were changed outside of evcc
func ResetRollup() error {
	return resetDaily(nil)
}

// writeDays runs a change to the slots or tariffs in [from,to). Only for
// completed days it waits for a running rollup and drops their rolled up rows.
func writeDays(from, to time.Time, change func(tx *gorm.DB) error) error {
	// a day that completes while the change runs counts as well
	if !from.Before(dailyFinal(time.Now().Add(tariff.SlotDuration))) {
		return db.Instance.Transaction(change)
	}

	dailyMu.Lock()
	defer dailyMu.Unlock()

	if err := db.Instance.Transaction(func(tx *gorm.DB) error {
		if err := change(tx); err != nil {
			return err
		}
		return deleteDaily(tx, dayStart(from).Unix(), to.Unix())
	}); err != nil {
		return err
	}

	requestRollup()
	return nil
}

// requestRollup wakes up Rollup
func requestRollup() {
	select {
	case dailyRequest <- struct{}{}:
	default: // already requested
	}
}

// Rollup rolls up the completed days: at start, when the next day completes
// and after a change to completed days. It blocks and is the only writer.
func Rollup() {
	for {
		now := time.Now()
		err := rollupDays(now)

		// the current day is complete, checked hourly in case the clock was set
		wait := min(time.Until(nextDay(dailyFinal(now)).Add(dailyDelay)), time.Hour)
		if err != nil {
			log.ERROR.Printf("history rollup: %v", err)
			wait = tariff.SlotDuration
		}

		select {
		case <-dailyRequest:
		case <-time.After(wait):
		}
	}
}

// rollupDays sums up the days completed at now that are not rolled up
func rollupDays(now time.Time) error {
	days, err := missingDays(dailyFinal(now))
	if err != nil {
		return err
	}

	for chunk := range slices.Chunk(days, dailyChunk) {
		if err := rollupChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// missingDays lists the days from the first slot up to final that are not rolled up
func missingDays(final time.Time) ([][2]int64, error) {
	first, _, err := slotRange()
	if err != nil || first == 0 {
		return nil, err
	}

	var rolled []int64
	if err := db.Instance.Model(new(flowDay)).Order("day").Pluck("day", &rolled).Error; err != nil {
		return nil, err
	}

	days := buckets(dayStart(time.Unix(first, 0)), final, "day")
	return slices.DeleteFunc(days, func(d [2]int64) bool {
		_, ok := slices.BinarySearch(rolled, d[0])
		return ok
	}), nil
}

// rollupChunk sums up the given days and stores them
func rollupChunk(days [][2]int64) error {
	dailyMu.Lock()
	defer dailyMu.Unlock()

	sums, err := sumEnergy(days, false, nil)
	if err != nil {
		return err
	}

	starts := make([]int64, 0, len(days))
	flows := make([]flowDay, 0, len(days))
	for _, d := range days {
		starts = append(starts, d[0])
		acc, err := accumulate(time.Unix(d[0], 0), time.Unix(d[1], 0))
		if err != nil {
			return err
		}
		// most entities are idle on a given day
		maps.DeleteFunc(acc.Entities, func(_ int, e *flowEntity) bool {
			return e.Metered == 0 && e.Flows == [3]flowSum{}
		})
		data, err := json.Marshal(acc)
		if err != nil {
			return err
		}
		flows = append(flows, flowDay{Day: d[0], Data: string(data)})
	}

	return db.Instance.Transaction(func(tx *gorm.DB) error {
		// sums left without their flow row would fail the insert for good
		if err := tx.Where("day IN ?", starts).Delete(new(energySum)).Error; err != nil {
			return err
		}
		if len(sums) > 0 {
			if err := tx.CreateInBatches(sums, 1000).Error; err != nil {
				return err
			}
		}
		return tx.Create(flows).Error
	})
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
	join := "LEFT JOIN meters_daily d ON d.day = f.day"
	var args []any
	if ids != nil {
		join += " AND d.meter IN ?"
		args = append(args, ids)
	}

	// one statement, so a day comes with all of its rows or not at all. A
	// day without energy has meter 0.
	rows, err := scanEnergy(db.Instance.Table("meters_flows_daily f").
		Select("f.day, COALESCE(d.meter, 0), 0, COALESCE(d.energy, 0), COALESCE(d.return_energy, 0), NULL, d.cost, d.priced_energy, d.return_cost, d.priced_return_energy").
		Joins(join, args...).
		Where("f.day >= ? AND f.day < ?", from.Unix(), to.Unix()))
	if err != nil {
		return nil, err
	}

	rolled := make(map[int64]bool)
	for _, r := range rows {
		rolled[r.Start] = true
	}

	// a day that is only partly inside the period is summed from its slots
	use := make(map[int64]bool)
	var rest [][2]int64
	for _, d := range buckets(from, to, "day") {
		if rolled[d[0]] && wholeDay(d) {
			use[d[0]] = true
		} else {
			rest = append(rest, d)
		}
	}

	res := slices.DeleteFunc(rows, func(s energySum) bool {
		return s.Meter == 0 || !use[s.Start]
	})

	slots, err := sumEnergy(rest, false, ids)
	return append(res, slots...), err
}

// dailyFlow attributes [from,to). Rolled up days are read from the rollup,
// the rest is attributed from the slots. Zero bounds are open ended.
func dailyFlow(from, to time.Time) (*flowAcc, error) {
	from, to, err := closeBounds(from, to, slotRange)
	if err != nil {
		return nil, err
	}

	var rows []flowDay
	if err := db.Instance.Where("day >= ? AND day < ?", from.Unix(), to.Unix()).Find(&rows).Error; err != nil {
		return nil, err
	}

	rolled := make(map[int64]string, len(rows))
	for _, r := range rows {
		rolled[r.Day] = r.Data
	}

	acc := &flowAcc{Entities: make(map[int]*flowEntity)}

	// the slots in between the rolled up days
	slots := func(start, end int64) error {
		if start == end {
			return nil
		}
		part, err := accumulate(time.Unix(start, 0), time.Unix(end, 0))
		if err != nil {
			return err
		}
		acc.merge(part)
		return nil
	}

	gap := from.Unix()
	for _, d := range buckets(from, to, "day") {
		data, ok := rolled[d[0]]
		if !ok || !wholeDay(d) {
			continue
		}

		if err := slots(gap, d[0]); err != nil {
			return nil, err
		}
		gap = d[1]

		var day flowAcc
		if err := json.Unmarshal([]byte(data), &day); err != nil {
			return nil, err
		}
		acc.merge(&day)
	}

	return acc, slots(gap, to.Unix())
}
