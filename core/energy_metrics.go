package core

import "math"

// EnergyMetrics calculates stats about the charged energy and gives you details about price or co2s
type EnergyMetrics struct {
	totalKWh          float64 // Total amount of energy used (kWh)
	solarKWh          float64 // Self-produced energy energy (kWh)
	price             float64 // Total cost (Currency)
	co2               float64 // Amount of emitted CO2 (gCO2eq)
	currentGreenShare float64 // Current share of solar energy of site (0-1)
	currentPrice      float64 // Current price per kWh
	currentCo2        float64 // Current co2 emissions
}

func NewEnergyMetrics() *EnergyMetrics {
	em := &EnergyMetrics{}
	em.Reset()

	return em
}

// SetEnvironment updates site information like solar share, price, co2 for use in later calculations
func (em *EnergyMetrics) SetEnvironment(greenShare float64, effPrice float64, effCo2 float64) {
	em.currentGreenShare = greenShare
	em.currentPrice = effPrice
	em.currentCo2 = effCo2
}

// Update sets the a new value for the total amount of charged energy and updated metrics based on enviroment values
func (em *EnergyMetrics) Update(chargedKWh float64) {
	added := chargedKWh - em.totalKWh
	// nothing changed
	if added == 0 {
		return
	}
	em.totalKWh = chargedKWh
	em.solarKWh += added * em.currentGreenShare
	em.price += added * em.currentPrice
	em.co2 += added * em.currentCo2
}

// Reset sets all calculations to initial values
func (em *EnergyMetrics) Reset() {
	em.totalKWh = 0
	em.solarKWh = 0
	em.price = 0
	em.co2 = 0
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
func (em *EnergyMetrics) Price() float64 {
	if em.totalKWh == 0 {
		return 0
	}
	return em.price
}

// PricePerKWh returns the average energy price in Currency
func (em *EnergyMetrics) PricePerKWh() float64 {
	if em.totalKWh == 0 {
		return 0
	}
	return em.price / em.totalKWh
}

// Co2PerKWh returns the average co2 emissions per kWh
func (em *EnergyMetrics) Co2PerKWh() float64 {
	if em.totalKWh == 0 {
		return 0
	}
	return em.co2 / em.totalKWh
}

// Publish publishes metrics with a given prefix
func (em *EnergyMetrics) Publish(prefix string, p publisher) {
	p.publish(prefix+"Energy", em.TotalWh())
	p.publish(prefix+"SolarPercentage", em.SolarPercentage())
	if v := em.PricePerKWh(); !math.IsNaN(v) {
		p.publish(prefix+"PricePerKWh", v)
	}
	if v := em.Price(); !math.IsNaN(v) {
		p.publish(prefix+"Price", v)
	}
	if v := em.Co2PerKWh(); !math.IsNaN(v) {
		p.publish(prefix+"Co2PerKWh", v)
	}
}
