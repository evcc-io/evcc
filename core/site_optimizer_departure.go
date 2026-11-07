package core

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
)

// departureHalfLife is the departure assumption without history: a connected vehicle
// leaves within 24h with a 50% chance, at a constant rate
const departureHalfLife = 24 * time.Hour

// departurePrior weighs departureHalfLife against the observed sessions of a slot, in
// sessions. Keeps a slot seen once or twice from swinging to all or nothing.
const departurePrior = 2.0

// departureSlotsPerDay are the time of day buckets, kept apart for weekdays and weekends
const departureSlotsPerDay = int(24 * time.Hour / tariff.SlotDuration)

func departureBucket(ts time.Time) int {
	ts = ts.Local()
	slot := (ts.Hour()*60 + ts.Minute()) / int(tariff.SlotDuration/time.Minute)
	if wd := ts.Weekday(); wd == time.Saturday || wd == time.Sunday {
		slot += departureSlotsPerDay
	}
	return slot
}

// departureProfile returns the probability that the vehicle leaves during each of the minLen
// slots starting at now. The hazard per time of day is the share of past sessions still
// connected at the slot that ended in it, blended with the constant hazard of departureHalfLife
// by departurePrior pseudo sessions. Without history that leaves the constant hazard alone.
func departureProfile(now time.Time, history session.Sessions, minLen int) []float32 {
	var ends, active [2 * departureSlotsPerDay]float64
	for _, s := range history {
		if !s.Finished.After(s.Created) {
			continue
		}
		for ts := s.Created.Truncate(tariff.SlotDuration).Add(tariff.SlotDuration); !ts.After(s.Finished); ts = ts.Add(tariff.SlotDuration) {
			active[departureBucket(ts)]++
		}
		if ts := s.Finished.Truncate(tariff.SlotDuration); ts.After(s.Created) {
			ends[departureBucket(ts)]++
		}
	}

	prior := 1 - math.Pow(0.5, float64(tariff.SlotDuration)/float64(departureHalfLife))

	res := make([]float32, minLen)
	survive := 1.0
	eos := now.Truncate(tariff.SlotDuration).Add(tariff.SlotDuration)
	for i := range res {
		start := eos.Add(time.Duration(i-1) * tariff.SlotDuration)
		b := departureBucket(start)
		hazard := (ends[b] + departurePrior*prior) / (active[b] + departurePrior)
		if i == 0 {
			// the first slot is already running
			hazard *= float64(eos.Sub(now)) / float64(tariff.SlotDuration)
		}
		res[i] = float32(survive * hazard)
		survive -= survive * hazard
	}

	return res
}

// departure returns the departure probabilities of the connected vehicle per slot
func (site *Site) departure(lp loadpoint.API, minLen int) []float32 {
	var history session.Sessions
	if title := lp.GetVehicle().GetTitle(); title != "" && db.Instance != nil {
		var err error
		if history, err = session.Completed(db.Instance, title); err != nil {
			site.log.WARN.Printf("optimizer: departure history: %v", err)
		}
	}

	return departureProfile(time.Now(), history, minLen)
}
