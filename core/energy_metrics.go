package core

// EnergyMetrics calculates stats about the charged energy and gives you details about price or co2s
type EnergyMetrics struct {
	totalKWh          float64  // Total amount of energy used (kWh)
	solarKWh          float64  // Self-produced energy (kWh)
	price             *float64 // Total cost (Currency)
	priceSaved        *float64 // Total saved cost (Currency)
	co2               *float64 // Amount of emitted CO2 (gCO2eq)
	co2Saved          *float64 // Amount of saved CO2 (gCO2eq)
	currentGreenShare float64  // Current share of solar energy of site (0-1)
	currentPrice      *float64 // Current price per kWh
	referencePrice    *float64 // Reference price per kWh
	currentCo2        *float64 // Current co2 emissions
	referenceCo2      *float64 // Reference co2 emissions
}

func NewEnergyMetrics() *EnergyMetrics {
	em := &EnergyMetrics{}
	em.Reset()

	return em
}

// SetEnvironment updates site information like solar share, price, co2 for use in later calculations
func (em *EnergyMetrics) SetEnvironment(greenShare float64, effPrice, refPrice, effCo2, refCo2 *float64) {
	em.currentGreenShare = greenShare
	em.currentPrice = effPrice
	em.referencePrice = refPrice
	em.currentCo2 = effCo2
	em.referenceCo2 = refCo2
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
	em.solarKWh += addedGreen
	if em.currentPrice != nil {
		if em.price == nil {
			em.price = new(float64)
			em.priceSaved = new(float64)
		}
		*em.price += *em.currentPrice * added
		*em.priceSaved = (*em.referencePrice - *em.currentPrice) * added
	}
	if em.currentCo2 != nil {
		if em.co2 == nil {
			em.co2 = new(float64)
			em.co2Saved = new(float64)
		}
		*em.co2 += *em.currentCo2 * added
		*em.co2Saved = (*em.referenceCo2 - *em.currentCo2) * added
	}
	return added, addedGreen
}

// Reset sets all calculations to initial values
func (em *EnergyMetrics) Reset() {
	em.totalKWh = 0
	em.solarKWh = 0
	em.price = nil
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

// Price returns the total energy price in Currency
func (em *EnergyMetrics) Price() *float64 {
	if em.totalKWh == 0 || em.price == nil {
		return nil
	}
	return em.price
}

// PriceSaved return the total amount of saved cost in Currency
func (em *EnergyMetrics) PriceSaved() *float64 {
	if em.totalKWh == 0 || em.priceSaved == nil {
		return nil
	}
	return em.priceSaved
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

// Co2Saved returns the total amount of saved co2 emissions in gCO2eq
func (em *EnergyMetrics) Co2Saved() *float64 {
	if em.totalKWh == 0 || em.co2Saved == nil {
		return nil
	}
	return em.co2Saved
}

// Publish publishes metrics with a given prefix
func (em *EnergyMetrics) Publish(prefix string, p publisher) {
	p.publish(prefix+"Energy", em.TotalWh())
	p.publish(prefix+"SolarPercentage", em.SolarPercentage())
	p.publish(prefix+"PricePerKWh", em.PricePerKWh())
	p.publish(prefix+"Price", em.Price())
	p.publish(prefix+"PriceSaved", em.PriceSaved())
	p.publish(prefix+"Co2PerKWh", em.Co2PerKWh())
	p.publish(prefix+"Co2Saved", em.Co2Saved())
}
