package core

import (
	"sort"
)

func (site *Site) priorityPowerDistribution(totalPriority int, totalChargePower float64, availablePower float64, cheapRate bool, batteryBuffered bool) {
	sort.Slice(site.loadpoints, func(i, j int) bool {
		return site.loadpoints[i].chargePriority > site.loadpoints[j].chargePriority
	})
	var powerRest float64 = availablePower
	var powerSliceNotUsed float64 = 0
	for _, lp := range site.loadpoints {
		if lp.chargePriority > 0 {
			var powerSlice = availablePower*float64(lp.chargePriority)/float64(totalPriority) + powerSliceNotUsed
			if powerRest > 0 {
				powerRest = 0
			}
			if powerRest > powerSlice {
				// then higer priority Loadpoint needed more power as the powerSlice was calculated for it to start charging
				powerSlice = powerRest
			}
			lp.log.DEBUG.Printf("priority of the loadpoint: %.0f%% - Slice: %.2f - Rest: %.2f", float64(lp.chargePriority)/float64(totalPriority)*100, powerSlice, powerRest)
			if powerSlice-lp.GetMinPowerRequried() > -lp.GetMinPower()*float64(lp.activePhases()) {
				//lp.log.DEBUG.Printf("priority not enoght for loadpoint : %.2f + %.2f > %.2f", powerSlice, -lp.GetMinPowerRequried(), -lp.GetMinPower()*float64(lp.activePhases()))
				if powerRest < -lp.GetMinPower()*float64(lp.activePhases()) {
					x := powerRest + lp.GetChargePower()
					lp.log.DEBUG.Printf("priority rest update: %.2f", x)
					lp.Update(x, cheapRate, batteryBuffered)
					if totalChargePower > 0 && lp.GetChargePower() <= 0 {
						lp.elapsePVTimer()
					}
					powerRest += lp.GetMaxPower()
				} else {
					// ToDo: powerRest don't fit for a start but could be added to an ealier Loadpoint
					powerSliceNotUsed += powerSlice
					lp.log.DEBUG.Printf("priority not enoght power - needed: %.2f", -lp.GetMinPower()*float64(lp.activePhases()))
					if lp.GetChargePower() > 0 && powerRest == availablePower {
						lp.elapsePVTimer()
					}
					lp.Update(lp.GetChargePower(), cheapRate, batteryBuffered)
				}
			} else {
				x := powerSlice - lp.GetMinPowerRequried() + lp.GetChargePower()
				lp.log.DEBUG.Printf("priority power update: %.2f", x)
				lp.Update(x, cheapRate, batteryBuffered)
				if totalChargePower > 0 && lp.GetChargePower() <= 0 {
					lp.elapsePVTimer()
				}
				powerRest -= powerSlice
			}
		} else {
			x := powerRest - lp.GetMinPowerRequried() + lp.GetChargePower()
			lp.log.DEBUG.Printf("no priority update : %.2f", x)
			lp.Update(x, cheapRate, batteryBuffered)
			powerRest = 0
		}
		if powerRest < 0 {
			// ToDo: powerRest don't fit for a start but could be added to an ealier Loadpoint
			site.log.DEBUG.Printf("priority power not provided: %.2f", powerRest)
		}
	}
}
