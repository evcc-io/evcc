package core

import (
	"sort"
)

func (site *Site) priorityPowerDistribution(totalPriority int, availablePower float64, cheapRate bool, batteryBuffered bool) {
	sort.Slice(site.loadpoints, func(i, j int) bool {
		return site.loadpoints[i].chargePriority > site.loadpoints[j].chargePriority
	})
	var powerRest float64 = availablePower
	var powerSliceNotUsed float64 = 0
	for _, lp := range site.loadpoints {
		if lp.chargePriority > 0 {
			lp.log.DEBUG.Printf("priority of the loadpoint : %.0f%%", float64(lp.chargePriority)/float64(totalPriority)*100)
			var powerSlice = availablePower*float64(lp.chargePriority)/float64(totalPriority) + powerSliceNotUsed
			if powerRest > 0 {
				powerRest = 0
			}
			if powerRest > powerSlice {
				// then higer priority Loadpoint needed more power as the powerSlice was calculated for it
				powerSlice = powerRest
			}
			if powerSlice-lp.GetMinPowerRequried() > -lp.GetMinPower()*float64(lp.activePhases()) {
				lp.log.DEBUG.Printf("priority not enoght for loadpoint : %.2f + %.2f > %.2f", powerSlice, -lp.GetMinPowerRequried(), -lp.GetMinPower()*float64(lp.activePhases()))
				if powerRest < -lp.GetMinPower()*float64(lp.activePhases()) {
					lp.log.DEBUG.Printf("priority powerRest : %.2f : powerSlice :  %.2f ", powerRest, powerSlice)
					lp.Update(powerRest+lp.GetChargePower(), cheapRate, batteryBuffered)
					powerRest += lp.GetMaxPower()
				} else {
					// ToDo: powerRest don't fit for a start but could be added to an ealier Loadpoint
					powerSliceNotUsed += powerSlice
					lp.log.DEBUG.Printf("priority not enoght for loadpoint : %.2f < %.2f", powerRest, -lp.GetMinPower()*float64(lp.activePhases()))
					if lp.chargePower > 0 {
						lp.elapsePVTimer()
					}
					lp.Update(lp.GetChargePower(), cheapRate, batteryBuffered)
				}
			} else {
				lp.log.DEBUG.Printf("priority power for loadpoint : %.2f - %.2f > %.2f ", powerSlice, -lp.GetMinPowerRequried(), -lp.GetMinPower()*float64(lp.activePhases()))
				lp.Update(powerSlice-lp.GetMinPowerRequried()+lp.GetChargePower(), cheapRate, batteryBuffered)
				powerRest -= powerSlice
			}
		} else {
			lp.log.DEBUG.Printf("no priority for loadpoint : %.2f - %.2f", powerRest, -lp.GetMinPowerRequried())
			lp.Update(powerRest-lp.GetMinPowerRequried()+lp.GetChargePower(), cheapRate, batteryBuffered)
			powerRest = 0
		}
		if powerRest < 0 {
			// ToDo: powerRest don't fit for a start but could be added to an ealier Loadpoint
			site.log.DEBUG.Printf("power not provided to any loadpoint : %.2f", powerRest)
		}
	}
}
