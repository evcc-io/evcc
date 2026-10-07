package metrics

import (
	"time"

	"github.com/evcc-io/evcc/db"
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
	Flows []Flow    `json:"flows"`
	Cost  *FlowCost `json:"cost,omitempty"`
	Co2   *FlowCo2  `json:"co2,omitempty"`
}

// flowSlot holds one 15min slot with all groups pivoted into columns
type flowSlot struct {
	PV, GridIn, GridOut, BatIn, BatOut, Home, Loadpoint float64
	Grid, FeedIn, Co2                                   *float64
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

// QueryFlow attributes the energy of [from,to) from sources to sinks per 15min
// slot and sums the result. Zero bounds are open ended.
func QueryFlow(from, to time.Time) (FlowResult, error) {
	tx := db.Instance.Table("meters m").
		Select(`
			COALESCE(SUM(CASE WHEN e."group" = 'pv' THEN m.energy END), 0) AS pv,
			COALESCE(SUM(CASE WHEN e."group" = 'grid' THEN m.energy END), 0) AS grid_in,
			COALESCE(SUM(CASE WHEN e."group" = 'grid' THEN m.return_energy END), 0) AS grid_out,
			COALESCE(SUM(CASE WHEN e."group" = 'battery' THEN m.energy END), 0) AS bat_in,
			COALESCE(SUM(CASE WHEN e."group" = 'battery' THEN m.return_energy END), 0) AS bat_out,
			COALESCE(SUM(CASE WHEN e."group" = 'home' THEN m.energy END), 0) AS home,
			COALESCE(SUM(CASE WHEN e."group" = 'loadpoint' THEN m.energy END), 0) AS loadpoint,
			t.grid, t.feedin AS feed_in, t.co2`).
		Joins("JOIN entities e ON m.meter = e.id").
		Joins("LEFT JOIN tariffs t ON t.ts = m.ts").
		Where(`e."group" IN ?`, []string{PV, Grid, Battery, Home, Loadpoint}).
		Group("m.ts")

	if !from.IsZero() {
		tx = tx.Where("m.ts >= ?", from.Unix())
	}
	if !to.IsZero() {
		tx = tx.Where("m.ts < ?", to.Unix())
	}

	// stream slots to keep memory flat for long periods
	rows, err := tx.Rows()
	if err != nil {
		return FlowResult{}, err
	}
	defer rows.Close()

	sums := make(map[[2]string]float64)
	costs := make(map[[2]string]float64)
	priced := make(map[[2]string]float64)
	var cost FlowCost
	var co2 FlowCo2
	var costSlots, co2Slots float64

	for rows.Next() {
		var s flowSlot
		if err := db.Instance.ScanRows(rows, &s); err != nil {
			return FlowResult{}, err
		}

		// energy that reached home and loadpoints, and the grid's part of it.
		// Consumption a slot's sources cannot cover (meter mismatch, coarse
		// counters) stays out of the cost model on both sides.
		var consumption, gridToConsumption float64
		attribute(s, func(from, to string, energy float64) {
			sums[[2]string{from, to}] += energy
			if to != Home && to != Loadpoint {
				return
			}
			if s.Grid != nil {
				key := [2]string{from, to}
				priced[key] += energy
				if from == Grid {
					costs[key] += energy * *s.Grid
				} else if s.FeedIn != nil {
					costs[key] += energy * *s.FeedIn
				}
			}
			consumption += energy
			if from == Grid {
				gridToConsumption += energy
			}
		})

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

	if err := rows.Err(); err != nil {
		return FlowResult{}, err
	}

	var res FlowResult
	for _, from := range flowSources {
		for _, to := range flowSinks {
			key := [2]string{from, to}
			if e := roundEnergy(sums[key]); e > 0 {
				res.Flows = append(res.Flows, Flow{From: from, To: to, Energy: e, Cost: costs[key], PricedEnergy: priced[key]})
			}
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
