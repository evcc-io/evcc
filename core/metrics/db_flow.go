package metrics

import (
	"time"

	"github.com/evcc-io/evcc/db"
)

// flow sinks; sources reuse the group names PV, Battery, Grid
const Export = "export"

// Flow is the energy attributed from one source to one sink over the queried period.
type Flow struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Energy float64 `json:"energy"` // kWh
}

// FlowCost sums the grid tariff over the slots that carry a grid price.
// Consumption is what home and loadpoints cost with grid energy at the grid
// price and self-produced energy at the feed-in price (lost revenue), Baseline
// what the same energy would have cost at the average grid price.
type FlowCost struct {
	Import            float64 `json:"import"`
	Export            float64 `json:"export"`
	ImportEnergy      float64 `json:"importEnergy"` // kWh priced
	ExportEnergy      float64 `json:"exportEnergy"` // kWh priced
	Consumption       float64 `json:"consumption"`
	ConsumptionEnergy float64 `json:"consumptionEnergy"` // kWh priced
	Home              float64 `json:"home"`              // consumption share of home only
	HomeEnergy        float64 `json:"homeEnergy"`        // kWh priced
	Baseline          float64 `json:"baseline"`
	AvgGrid           float64 `json:"avgGrid"`
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
	var cost FlowCost
	var co2 FlowCo2
	var costSlots, co2Slots float64

	for rows.Next() {
		var s flowSlot
		if err := db.Instance.ScanRows(rows, &s); err != nil {
			return FlowResult{}, err
		}

		// energy that reached home and loadpoints, split by grid and self-produced.
		// Consumption a slot's sources cannot cover (meter mismatch, coarse
		// counters) stays out of the cost model on both sides.
		var gridToConsumption, greenToConsumption, gridToHome, greenToHome float64
		attribute(s, func(from, to string, energy float64) {
			sums[[2]string{from, to}] += energy
			if to != Home && to != Loadpoint {
				return
			}
			if from == Grid {
				gridToConsumption += energy
			} else {
				greenToConsumption += energy
			}
			if to == Home {
				if from == Grid {
					gridToHome += energy
				} else {
					greenToHome += energy
				}
			}
		})

		if s.Grid != nil {
			cost.Import += s.GridIn * *s.Grid
			cost.ImportEnergy += s.GridIn
			cost.AvgGrid += *s.Grid
			costSlots++
			cost.ConsumptionEnergy += gridToConsumption + greenToConsumption
			cost.Consumption += gridToConsumption * *s.Grid
			cost.HomeEnergy += gridToHome + greenToHome
			cost.Home += gridToHome * *s.Grid
			if s.FeedIn != nil {
				cost.Export += s.GridOut * *s.FeedIn
				cost.ExportEnergy += s.GridOut
				cost.Consumption += greenToConsumption * *s.FeedIn
				cost.Home += greenToHome * *s.FeedIn
			}
		}

		if s.Co2 != nil {
			co2.Consumption += gridToConsumption * *s.Co2 / 1e3
			co2.ConsumptionEnergy += gridToConsumption + greenToConsumption
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
			if e := roundEnergy(sums[[2]string{from, to}]); e > 0 {
				res.Flows = append(res.Flows, Flow{From: from, To: to, Energy: e})
			}
		}
	}

	if costSlots > 0 {
		cost.AvgGrid /= costSlots
		cost.Baseline = cost.AvgGrid * cost.ConsumptionEnergy
		res.Cost = &cost
	}

	if co2Slots > 0 {
		co2.AvgCo2 /= co2Slots
		co2.Baseline = co2.AvgCo2 * co2.ConsumptionEnergy / 1e3
		res.Co2 = &co2
	}

	return res, nil
}
