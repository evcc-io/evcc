package core

import "time"

// EnergyMetrics calculates stats about the charged energy and gives you details about price or co2s
type EnergyMetrics struct {
	totalKWh          float64  // Total amount of energy used (kWh)
	solarKWh          float64  // Self-produced energy (kWh)
	price             *float64 // Total cost (Currency)
	co2               *float64 // Amount of emitted CO2 (gCO2eq)
	currentGreenShare float64  // Current share of solar energy of site (0-1)
	currentPrice      *float64 // Current price per kWh
	currentCo2        *float64 // Current co2 emissions

	// The environment above is a *snapshot*, taken when it is set. It may only be applied to energy that
	// was actually consumed while it was valid — see Sample.
	sampledKWh   float64   // energy sampled since the last Update (kWh)
	sampledGreen float64   // the green part of it (kWh)
	sampledCost  float64   // its cost (Currency)
	sampledCo2   float64   // its CO2 (gCO2eq)
	lastSample   time.Time // time of the previous Sample
}

// SetEnvironment updates site information like solar share, price, co2 for use in later calculations
func (em *EnergyMetrics) SetEnvironment(greenShare float64, effPrice, effCo2 *float64) {
	em.currentGreenShare = greenShare
	em.currentPrice = effPrice
	em.currentCo2 = effCo2
}

// Sample integrates the loadpoint's power into the pending energy window, weighted by the environment that
// is valid at this instant. Call it once per poll, before Update.
//
// Why this exists: `totalKWh` only ever moves when the device's counter moves, and a counter can move long
// after the energy was consumed. A **self-metering** device (heat pump, water heater: `Heating` +
// `IntegratedDevice`) reports its counter in bursts — the Buderus cloud counter advances every ~11 min
// regardless of what the pump does — so attributing the whole step to the environment read at that one
// instant values up to eleven minutes of consumption with a single sample. Measured 2026-11-04: a pump that
// ran all night on imported power booked its day at 98 % solar, and its cost at the feed-in tariff instead
// of the grid tariff, because the share was read at a moment when the battery happened to be discharging.
//
// Sampling turns that snapshot into the energy-weighted mean over the interval the counter step covers:
// ∫P·share dt / ∫P dt, which is the same number where the environment is constant and the right number
// where it is not. Elapsed time is passed in rather than read from a clock, so a caller with a fake clock
// and a test are the same thing.
//
// The window is not reset here: it accumulates until Update consumes it.
func (em *EnergyMetrics) Sample(power float64, now time.Time) {
	prev := em.lastSample
	em.lastSample = now

	// first sample, clock stepped back, or a resting device: nothing to weight
	if prev.IsZero() || power <= 0 {
		return
	}

	elapsed := now.Sub(prev)
	if elapsed <= 0 {
		return
	}

	kWh := power * elapsed.Hours() / 1e3
	em.sampledKWh += kWh
	em.sampledGreen += kWh * em.currentGreenShare

	if em.currentPrice != nil {
		em.sampledCost += kWh * *em.currentPrice
	}
	if em.currentCo2 != nil {
		em.sampledCo2 += kWh * *em.currentCo2
	}
}

// environment returns the environment to apply to the next energy step, averaged over the sampled window.
// Without a usable window it is the current snapshot — a device that was asleep while its counter caught up
// gives no better information, and a caller that never samples (an EV meter that moves on every poll) keeps
// exactly the previous behaviour.
func (em *EnergyMetrics) environment() (greenShare float64, price, co2 *float64) {
	if em.sampledKWh <= 0 {
		return em.currentGreenShare, em.currentPrice, em.currentCo2
	}

	greenShare = em.sampledGreen / em.sampledKWh
	if em.currentPrice != nil {
		v := em.sampledCost / em.sampledKWh
		price = &v
	}
	if em.currentCo2 != nil {
		v := em.sampledCo2 / em.sampledKWh
		co2 = &v
	}

	return greenShare, price, co2
}

// Update sets the a new value for the total amount of charged energy and updated metrics based on environment values.
// It returns the added total and green energy.
func (em *EnergyMetrics) Update(chargedKWh float64) (float64, float64) {
	added := chargedKWh - em.totalKWh
	// nothing changed or invalid lower value
	if added <= 0 {
		return 0, 0
	}
	em.totalKWh = chargedKWh

	// the step is valued with the environment averaged over the interval it covers, not with the one read
	// at the moment the counter moved
	greenShare, pricePerKWh, co2PerKWh := em.environment()
	em.sampledKWh, em.sampledGreen, em.sampledCost, em.sampledCo2 = 0, 0, 0, 0

	addedGreen := added * greenShare
	em.solarKWh += addedGreen
	// optional values
	if pricePerKWh != nil {
		addedPrice := *pricePerKWh * added
		newPrice := addedPrice
		if em.price != nil {
			newPrice = *em.price + newPrice
		}
		em.price = &newPrice
	}
	if co2PerKWh != nil {
		addedCo2 := *co2PerKWh * added
		newCo2 := addedCo2
		if em.co2 != nil {
			newCo2 = *em.co2 + newCo2
		}
		em.co2 = &newCo2
	}
	return added, addedGreen
}

// Restore seeds the metrics from an already persisted session. A restart mid-day is routine, and without
// this the day row adopted from the database met an accumulator at zero: the next counter move was then
// booked as the *whole* day's energy, valued with that one moment's environment, overwriting the share of
// everything that had been booked correctly before the restart. Measured 2026-11-04 13:35: "Brauchwasser"
// stood at 6,82 kWh / 62,0 % and read 7,33 kWh / 18,4 % a quarter of an hour later — energy grew by 0,51
// kWh while the green energy *fell*, which no accumulator can do. Reusing the persisted numbers keeps the
// day's history instead of re-deriving it.
func (em *EnergyMetrics) Restore(chargedKWh, solarPercentage float64, pricePerKWh, co2PerKWh *float64) {
	if chargedKWh <= 0 {
		return
	}

	em.totalKWh = chargedKWh
	em.solarKWh = chargedKWh * solarPercentage / 100

	if pricePerKWh != nil {
		v := *pricePerKWh * chargedKWh
		em.price = &v
	}
	if co2PerKWh != nil {
		v := *co2PerKWh * chargedKWh
		em.co2 = &v
	}

	// the window restarts with the process: the downtime is not consumption we can attribute
	em.sampledKWh, em.sampledGreen, em.sampledCost, em.sampledCo2 = 0, 0, 0, 0
	em.lastSample = time.Time{}
}

// Reset sets all calculations to initial values
func (em *EnergyMetrics) Reset() {
	em.totalKWh = 0
	em.solarKWh = 0
	em.price = nil
	em.co2 = nil

	// a new session starts a new window: the next Sample only records its own instant
	em.sampledKWh = 0
	em.sampledGreen = 0
	em.sampledCost = 0
	em.sampledCo2 = 0
	em.lastSample = time.Time{}
}

// TotalWh returns the total energy in Wh
func (em *EnergyMetrics) TotalWh() float64 {
	return em.totalKWh * 1e3
}

// SolarPercentage returns the share of self-produced energy in percent
func (em *EnergyMetrics) SolarPercentage() float64 {
	if em.totalKWh == 0 {
		return 0
	}
	return 100 / em.totalKWh * em.solarKWh
}

// Price returns the total energy price in Currency
func (em *EnergyMetrics) Price() *float64 {
	if em.totalKWh == 0 || em.price == nil {
		return nil
	}
	return em.price
}

// PricePerKWh returns the average energy price in Currency
func (em *EnergyMetrics) PricePerKWh() *float64 {
	if em.totalKWh == 0 || em.price == nil {
		return nil
	}
	price := *em.price / em.totalKWh
	return &price
}

// Co2PerKWh returns the average co2 emissions per kWh
func (em *EnergyMetrics) Co2PerKWh() *float64 {
	if em.totalKWh == 0 || em.co2 == nil {
		return nil
	}
	co2 := *em.co2 / em.totalKWh
	return &co2
}

// Publish publishes metrics with a given prefix
func (em *EnergyMetrics) Publish(prefix string, p publisher) {
	p.publish(prefix+"Energy", em.TotalWh())
	p.publish(prefix+"SolarPercentage", em.SolarPercentage())
	p.publish(prefix+"PricePerKWh", em.PricePerKWh())
	p.publish(prefix+"Price", em.Price())
	p.publish(prefix+"Co2PerKWh", em.Co2PerKWh())
}
