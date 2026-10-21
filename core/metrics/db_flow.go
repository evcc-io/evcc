package metrics

import (
	"maps"
	"slices"
	"time"

	"github.com/evcc-io/evcc/db"
	"golang.org/x/sync/errgroup"
)

// flow sinks; sources reuse the group names PV, Battery, Grid
const Export = "export"

// Flow is the energy attributed from one source to one sink over the queried period.
type Flow struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	Energy       float64 `json:"energy"`                 // kWh
	Cost         float64 `json:"cost,omitempty"`         // home and loadpoint only
	PricedEnergy float64 `json:"pricedEnergy,omitempty"` // kWh with a price
}

// EntityFlows is what reached one consumer or loadpoint: per slot the sink's
// source split scaled to the entity's energy, priced at the slot's tariff.
type EntityFlows struct {
	Group   string `json:"group"`
	Title   string `json:"title"`
	Virtual bool   `json:"virtual,omitempty"` // home minus the tracked consumers
	Flows   []Flow `json:"flows"`
}

// FlowCost sums the grid tariff over the slots that carry a grid price. What
// home and loadpoints cost is on the flows that reached them.
type FlowCost struct {
	Import       float64 `json:"import"`
	Export       float64 `json:"export"`
	ImportEnergy float64 `json:"importEnergy"` // kWh with a price
	ExportEnergy float64 `json:"exportEnergy"` // kWh with a price
	AvgGrid      float64 `json:"avgGrid"`
}

// FlowCo2 sums the grid co2 intensity (kg) over the slots that carry a co2
// value. Consumption counts only the grid share that went to home and loadpoints.
type FlowCo2 struct {
	Consumption       float64 `json:"consumption"`
	ConsumptionEnergy float64 `json:"consumptionEnergy"` // kWh
	Baseline          float64 `json:"baseline"`
	AvgCo2            float64 `json:"avgCo2"` // g/kWh
}

// FlowResult is the source to sink energy attribution of a period.
type FlowResult struct {
	Flows    []Flow        `json:"flows"`
	Entities []EntityFlows `json:"entities,omitempty"`
	Cost     *FlowCost     `json:"cost,omitempty"`
	Co2      *FlowCo2      `json:"co2,omitempty"`
}

// flowSlot holds one 15min slot with all groups pivoted into columns
type flowSlot struct {
	Ts                                                  int64
	PV, GridIn, GridOut, BatIn, BatOut, Home, Loadpoint float64
	Grid, FeedIn, Co2                                   *float64
}

// price of energy from a source in this slot: grid at the grid price, own
// energy at the feed-in price it could have earned. Nothing is priced without a grid price.
func (s flowSlot) price(from string, energy float64) (cost, priced float64) {
	if s.Grid == nil {
		return 0, 0
	}
	if from == Grid {
		return energy * *s.Grid, energy
	}
	if s.FeedIn != nil {
		return energy * *s.FeedIn, energy
	}
	return 0, energy
}

// flowSum accumulates one flow over the period
type flowSum struct{ energy, cost, priced float64 }

func (f *flowSum) add(s flowSlot, from string, energy float64) {
	c, p := s.price(from, energy)
	f.energy += energy
	f.cost += c
	f.priced += p
}

var (
	flowSources = []string{PV, Battery, Grid}
	flowSinks   = []string{Home, Battery, Loadpoint, Export}
)

// attribute distributes the slot's sources onto its sinks in priority order:
// pv before battery before grid, home before battery charge before loadpoints
// before export. Mirrors the live greenShare rule. Any balance mismatch is dropped.
func attribute(s flowSlot, add func(from, to string, energy float64)) {
	src := map[string]float64{PV: s.PV, Battery: s.BatOut, Grid: s.GridIn}
	dst := map[string]float64{Home: s.Home, Battery: s.BatIn, Loadpoint: s.Loadpoint, Export: s.GridOut}

	for _, from := range flowSources {
		for _, to := range flowSinks {
			if from == to || (from == Grid && to == Export) {
				continue
			}
			if x := min(src[from], dst[to]); x > 0 {
				add(from, to, x)
				src[from] -= x
				dst[to] -= x
			}
		}
	}
}

// flowEntity is a consumer or loadpoint priced on its own
type flowEntity struct {
	entity
	flows [3]flowSum // per flowSources entry
}

// share gives the entity its energy in the sink's source split of the slot.
// Dividing by the metered sink leaves out the part its sources cannot cover.
func (e *flowEntity) share(s flowSlot, energy float64, sink [3]float64, metered float64) {
	if energy <= 0 || metered <= 0 {
		return
	}
	for i, from := range flowSources {
		e.flows[i].add(s, from, energy*sink[i]/metered)
	}
}

// slotEnergy is the energy of a consumer or loadpoint in one slot
type slotEnergy struct {
	*flowEntity
	energy float64
}

// entitySlots returns the energy of the given entities per slot. Without the
// entities join this reads their rows (the bulk of the table) about twice as
// fast as pivoting them along with the groups.
func entitySlots(entities map[int]*flowEntity, from, to time.Time) (map[int64][]slotEnergy, error) {
	res := make(map[int64][]slotEnergy)
	if len(entities) == 0 {
		return res, nil
	}

	tx := db.Instance.Table("meters").
		Select("ts, meter, energy").
		Where("meter IN ? AND energy <> 0", slices.Collect(maps.Keys(entities)))

	if !from.IsZero() {
		tx = tx.Where("ts >= ?", from.Unix())
	}
	if !to.IsZero() {
		tx = tx.Where("ts < ?", to.Unix())
	}

	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ts int64
		var id int
		var energy float64
		if err := rows.Scan(&ts, &id, &energy); err != nil {
			return nil, err
		}
		res[ts] = append(res[ts], slotEnergy{entities[id], energy})
	}

	return res, rows.Err()
}

// sinkSlots returns the source and home meters pivoted per slot, with the slot's tariff
func sinkSlots(from, to time.Time) ([]flowSlot, error) {
	tx := db.Instance.Table("meters m").
		Select(`
			m.ts,
			COALESCE(SUM(CASE WHEN e."group" = 'pv' THEN m.energy END), 0) AS pv,
			COALESCE(SUM(CASE WHEN e."group" = 'grid' THEN m.energy END), 0) AS grid_in,
			COALESCE(SUM(CASE WHEN e."group" = 'grid' THEN m.return_energy END), 0) AS grid_out,
			COALESCE(SUM(CASE WHEN e."group" = 'battery' THEN m.energy END), 0) AS bat_in,
			COALESCE(SUM(CASE WHEN e."group" = 'battery' THEN m.return_energy END), 0) AS bat_out,
			COALESCE(SUM(CASE WHEN e."group" = 'home' THEN m.energy END), 0) AS home,
			t.grid, t.feedin, t.co2`).
		Joins("JOIN entities e ON m.meter = e.id").
		Joins("LEFT JOIN tariffs t ON t.ts = m.ts").
		Where(`e."group" IN ?`, []string{PV, Grid, Battery, Home}).
		Group("m.ts")

	if !from.IsZero() {
		tx = tx.Where("m.ts >= ?", from.Unix())
	}
	if !to.IsZero() {
		tx = tx.Where("m.ts < ?", to.Unix())
	}

	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []flowSlot
	for rows.Next() {
		var s flowSlot
		// scanned by hand: gorm's reflective ScanRows dominates long periods
		if err := rows.Scan(&s.Ts, &s.PV, &s.GridIn, &s.GridOut, &s.BatIn, &s.BatOut, &s.Home, &s.Grid, &s.FeedIn, &s.Co2); err != nil {
			return nil, err
		}
		res = append(res, s)
	}

	return res, rows.Err()
}

// QueryFlow attributes the energy of [from,to) from sources to sinks per 15min
// slot and sums the result. Zero bounds are open ended.
func QueryFlow(from, to time.Time) (FlowResult, error) {
	// consumers and loadpoints are priced individually
	var ents []entity
	if err := db.Instance.Select(`id, "group", COALESCE(NULLIF(title,''), name) AS title`).
		Where(`"group" IN ?`, []string{Consumer, Loadpoint}).Order(`"group", title`).Find(&ents).Error; err != nil {
		return FlowResult{}, err
	}

	// meters sharing a title are one entity, as in the history
	entities := make(map[int]*flowEntity, len(ents))
	byTitle := make(map[[2]string]*flowEntity)
	list := make([]*flowEntity, 0, len(ents)+1)
	for _, e := range ents {
		key := [2]string{e.Group, e.Title}
		if byTitle[key] == nil {
			byTitle[key] = &flowEntity{entity: e}
			list = append(list, byTitle[key])
		}
		entities[e.Id] = byTitle[key]
	}

	// both reads are bound by page lookups and run side by side on WAL readers
	var perSlot map[int64][]slotEnergy
	var slots []flowSlot
	var eg errgroup.Group
	eg.Go(func() error {
		var err error
		perSlot, err = entitySlots(entities, from, to)
		return err
	})
	eg.Go(func() error {
		var err error
		slots, err = sinkSlots(from, to)
		return err
	})
	if err := eg.Wait(); err != nil {
		return FlowResult{}, err
	}

	// home minus the tracked consumers, per slot like the UI's residual
	others := &flowEntity{}
	others.Group = Consumer
	list = append(list, others)

	sums := make(map[[2]string]flowSum)
	var cost FlowCost
	var co2 FlowCo2
	var costSlots, co2Slots float64

	for _, s := range slots {
		// the slot's loadpoints add up to their sink
		slotEntities := perSlot[s.Ts]
		for _, se := range slotEntities {
			if se.Group == Loadpoint {
				s.Loadpoint += se.energy
			}
		}

		// energy that reached home and loadpoints, and the grid's part of it.
		// Consumption a slot's sources cannot cover (meter mismatch, coarse
		// counters) stays out of the cost model on both sides.
		var consumption, gridToConsumption float64
		var homeSink, lpSink [3]float64 // per flowSources entry
		attribute(s, func(from, to string, energy float64) {
			key := [2]string{from, to}
			f := sums[key]
			consumed := to == Home || to == Loadpoint
			if consumed {
				f.add(s, from, energy)
			} else {
				f.energy += energy
			}
			sums[key] = f
			if !consumed {
				return
			}
			consumption += energy
			if from == Grid {
				gridToConsumption += energy
			}
			i := slices.Index(flowSources, from)
			if to == Home {
				homeSink[i] += energy
			} else {
				lpSink[i] += energy
			}
		})

		var tracked float64
		for _, se := range slotEntities {
			if se.Group == Loadpoint {
				se.share(s, se.energy, lpSink, s.Loadpoint)
			} else {
				se.share(s, se.energy, homeSink, s.Home)
				tracked += se.energy
			}
		}
		others.share(s, max(0, s.Home-tracked), homeSink, s.Home)

		if s.Grid != nil {
			cost.Import += s.GridIn * *s.Grid
			cost.ImportEnergy += s.GridIn
			cost.AvgGrid += *s.Grid
			costSlots++
			if s.FeedIn != nil {
				cost.Export += s.GridOut * *s.FeedIn
				cost.ExportEnergy += s.GridOut
			}
		}

		if s.Co2 != nil {
			co2.Consumption += gridToConsumption * *s.Co2 / 1e3
			co2.ConsumptionEnergy += consumption
			co2.AvgCo2 += *s.Co2
			co2Slots++
		}
	}

	var res FlowResult
	for _, from := range flowSources {
		for _, to := range flowSinks {
			if f := sums[[2]string{from, to}]; roundEnergy(f.energy) > 0 {
				res.Flows = append(res.Flows, Flow{From: from, To: to, Energy: roundEnergy(f.energy), Cost: f.cost, PricedEnergy: f.priced})
			}
		}
	}

	for _, e := range list {
		sink := Home
		if e.Group == Loadpoint {
			sink = Loadpoint
		}
		ef := EntityFlows{Group: e.Group, Title: e.Title, Virtual: e == others}
		for i, from := range flowSources {
			f := e.flows[i]
			if energy := roundEnergy(f.energy); energy > 0 {
				ef.Flows = append(ef.Flows, Flow{From: from, To: sink, Energy: energy, Cost: f.cost, PricedEnergy: f.priced})
			}
		}
		if len(ef.Flows) > 0 {
			res.Entities = append(res.Entities, ef)
		}
	}

	if costSlots > 0 {
		cost.AvgGrid /= costSlots
		res.Cost = &cost
	}

	if co2Slots > 0 {
		co2.AvgCo2 /= co2Slots
		co2.Baseline = co2.AvgCo2 * co2.ConsumptionEnergy / 1e3
		res.Co2 = &co2
	}

	return res, nil
}
