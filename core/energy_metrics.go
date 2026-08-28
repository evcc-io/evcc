package core

// EnergyMetrics calculates stats about the charged energy and gives you details about price or co2s
type EnergyMetrics struct {
	totalKWh           float64  // Total amount of energy used (kWh)
	solarKWh           float64  // Self-produced energy (kWh)
	gridCost           *float64 // Cost of grid-imported energy (Currency)
	solarCost          *float64 // Opportunity cost of self-consumed solar energy, i.e. foregone feed-in revenue (Currency)
	co2                *float64 // Amount of emitted CO2 (gCO2eq)
	currentGreenShare  float64  // Current share of solar energy of site (0-1)
	currentGridPrice   *float64 // Current grid import price per kWh
	currentFeedInPrice *float64 // Current feed-in price per kWh, used to value self-consumed solar energy
	currentCo2         *float64 // Current co2 emissions
}

// SetEnvironment updates site information like solar share, grid and feed-in price, and co2 for use in later calculations
func (em *EnergyMetrics) SetEnvironment(greenShare float64, gridPrice, feedInPrice, effCo2 *float64) {
	em.currentGreenShare = greenShare
	em.currentGridPrice = gridPrice
	em.currentFeedInPrice = feedInPrice
	em.currentCo2 = effCo2
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
	addedGreen := added * em.currentGreenShare
	addedGrid := added - addedGreen
	em.solarKWh += addedGreen
	// optional values
	if em.currentGridPrice != nil {
		addedCost := *em.currentGridPrice * addedGrid
		newCost := addedCost
		if em.gridCost != nil {
			newCost = *em.gridCost + newCost
		}
		em.gridCost = &newCost
	}
	if em.currentFeedInPrice != nil {
		addedCost := *em.currentFeedInPrice * addedGreen
		newCost := addedCost
		if em.solarCost != nil {
			newCost = *em.solarCost + newCost
		}
		em.solarCost = &newCost
	}
	if em.currentCo2 != nil {
		addedCo2 := *em.currentCo2 * added
		newCo2 := addedCo2
		if em.co2 != nil {
			newCo2 = *em.co2 + newCo2
		}
		em.co2 = &newCo2
	}
	return added, addedGreen
}

// Reset sets all calculations to initial values
func (em *EnergyMetrics) Reset() {
	em.totalKWh = 0
	em.solarKWh = 0
	em.gridCost = nil
	em.solarCost = nil
	em.co2 = nil
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

// GridCost returns the cost of grid-imported energy in Currency
func (em *EnergyMetrics) GridCost() *float64 {
	if em.totalKWh == 0 || em.gridCost == nil {
		return nil
	}
	return em.gridCost
}

// SolarCost returns the opportunity cost of self-consumed solar energy in Currency,
// i.e. the feed-in revenue foregone by using the solar energy for charging instead of exporting it
func (em *EnergyMetrics) SolarCost() *float64 {
	if em.totalKWh == 0 || em.solarCost == nil {
		return nil
	}
	return em.solarCost
}

// Price returns the total energy price in Currency, the sum of grid cost and solar opportunity cost
func (em *EnergyMetrics) Price() *float64 {
	if em.totalKWh == 0 || (em.gridCost == nil && em.solarCost == nil) {
		return nil
	}
	var price float64
	if em.gridCost != nil {
		price += *em.gridCost
	}
	if em.solarCost != nil {
		price += *em.solarCost
	}
	return &price
}

// PricePerKWh returns the average energy price in Currency
func (em *EnergyMetrics) PricePerKWh() *float64 {
	price := em.Price()
	if em.totalKWh == 0 || price == nil {
		return nil
	}
	perKWh := *price / em.totalKWh
	return &perKWh
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
	p.publish(prefix+"GridCost", em.GridCost())
	p.publish(prefix+"SolarCost", em.SolarCost())
	p.publish(prefix+"Co2PerKWh", em.Co2PerKWh())
}
