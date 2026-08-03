package soc

// Anchor is the estimator's reference point: the last soc the vehicle itself
// reported, and the energy delivered at the charger since then.
//
// It is the anchor rather than the estimate that has to survive a restart.
// Soc() recomputes vehicleSoc from the anchor on every poll, so a restored
// vehicleSoc would be overwritten by the next call, while a restored anchor
// reproduces the same estimate for as long as the vehicle keeps reporting the
// value it was measured against.
type Anchor struct {
	Soc         float64 // vehicle-reported soc in %
	EnergySince float64 // energy delivered since, in Wh
}

// Anchor returns the estimator's current reference point.
func (s *Estimator) Anchor() Anchor {
	return Anchor{
		Soc:         s.prevSoc,
		EnergySince: s.chargedEnergy - s.prevChargedEnergy,
	}
}

// EnergyPerSocStep returns the estimator's Wh per soc percentage point.
func (s *Estimator) EnergyPerSocStep() float64 {
	return s.energyPerSocStep
}

// Restore seeds the estimator from a persisted anchor.
//
// chargedEnergy is the loadpoint's session counter the anchor is measured
// against. It is not necessarily zero: evcc may have restarted mid-session.
//
// Setting prevSoc is what makes this work at all. A fresh estimator has
// prevSoc 0, so the first poll would see socDelta != 0, take the rebase branch
// and discard everything restored here. With prevSoc seeded, a poll that still
// reads the same vehicle value takes the estimating branch and reproduces the
// stored estimate — and a poll that reads a genuinely changed value rebases,
// which is exactly the wanted behaviour: a real reading always wins.
//
// initialSoc and initialEnergy are deliberately left untouched; they anchor
// the in-session gradient learner, which is per-session by design.
func (s *Estimator) Restore(a Anchor, chargedEnergy float64) {
	s.prevSoc = a.Soc
	s.fetchedSoc = a.Soc
	s.chargedEnergy = max(chargedEnergy, 0)
	s.prevChargedEnergy = s.chargedEnergy - a.EnergySince

	// without a usable gradient there is no Wh/% to convert the energy into an
	// offset. Dividing by zero would yield +Inf, which min(_, 100) silently
	// turns into a plausible-looking full battery.
	if s.energyPerSocStep > 0 {
		s.vehicleSoc = min(a.Soc+a.EnergySince/s.energyPerSocStep, 100)
	} else {
		s.vehicleSoc = a.Soc
	}
}
