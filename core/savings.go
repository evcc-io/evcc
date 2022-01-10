package core

import (
	"math"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/tariff"
)

const DefaultGridPrice = 0.30
const DefaultFeedInPrice = 0.08

// publisher gives access to the site's publish function
type publisher interface {
	publish(key string, val interface{})
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Savings struct {
	clock                  clock.Clock
	tariffs                tariff.Tariffs
	started                time.Time // Boot time
	updated                time.Time // Time of last charged value update
	gridCharged            float64   // Grid energy charged since startup (kWh)
	gridCost               float64   // Running total of charged grid energy cost (e.g. EUR)
	selfConsumptionCharged float64   // Self-produced energy charged since startup (kWh)
	selfConsumptionCost    float64   // Running total of charged self-produced energy cost (e.g. EUR)
	lastGridPrice          float64   // Stores the last published grid price. Needed to detect price changes (Awattar, ..)
}

func NewSavings(tariffs tariff.Tariffs) *Savings {
	clock := clock.New()
	savings := &Savings{
		clock:   clock,
		tariffs: tariffs,
		started: clock.Now(),
		updated: clock.Now(),
	}

	return savings
}

func (s *Savings) Since() time.Duration {
	return time.Since(s.started)
}

func (s *Savings) SelfConsumptionPercent() float64 {
	if s.TotalCharged() == 0 {
		return 0
	}
	return s.selfConsumptionCharged / s.TotalCharged() * 100
}

func (s *Savings) TotalCharged() float64 {
	return s.gridCharged + s.selfConsumptionCharged
}

func (s *Savings) CostTotal() float64 {
	return s.gridCost + s.selfConsumptionCost
}

func (s *Savings) EffectivePrice() float64 {
	if s.TotalCharged() == 0 {
		return s.currentGridPrice()
	}
	return s.CostTotal() / s.TotalCharged()
}

func (s *Savings) SavingsAmount() float64 {
	return s.selfConsumptionCharged * (s.currentGridPrice() - s.currentFeedInPrice())
}

func (s *Savings) shareOfSelfProducedEnergy(gridPower, pvPower, batteryPower float64) float64 {
	batteryDischarge := math.Max(0, batteryPower)
	batteryCharge := math.Min(0, batteryPower) * -1
	pvConsumption := math.Min(pvPower, pvPower+gridPower-batteryCharge)

	gridImport := math.Max(0, gridPower)
	selfConsumption := math.Max(0, batteryDischarge+pvConsumption+batteryCharge)

	share := selfConsumption / (gridImport + selfConsumption)

	if math.IsNaN(share) {
		return 0
	}

	return share
}

func (s *Savings) currentGridPrice() float64 {
	if s.tariffs.Grid != nil {
		if gridPrice, err := s.tariffs.Grid.CurrentPrice(); err == nil {
			return gridPrice
		}
	}
	return DefaultGridPrice
}

func (s *Savings) currentFeedInPrice() float64 {
	if s.tariffs.FeedIn != nil {
		if gridPrice, err := s.tariffs.FeedIn.CurrentPrice(); err == nil {
			return gridPrice
		}
	}
	return DefaultFeedInPrice
}

func (s *Savings) Update(p publisher, gridPower, pvPower, batteryPower, chargePower float64) {
	// assume charge power as constant over the duration -> rough kWh estimate
	energyAdded := s.clock.Since(s.updated).Hours() * chargePower / 1e3
	s.updated = s.clock.Now()

	// nothing meaningfull changed, no need to update
	if energyAdded == 0 && s.lastGridPrice == s.currentGridPrice() {
		return
	}

	share := s.shareOfSelfProducedEnergy(gridPower, pvPower, batteryPower)

	addedSelfConsumption := energyAdded * share
	addedGrid := energyAdded - addedSelfConsumption

	s.gridCharged += addedGrid
	s.gridCost += addedGrid * s.currentGridPrice()
	s.selfConsumptionCharged += addedSelfConsumption
	s.selfConsumptionCost += addedSelfConsumption * s.currentFeedInPrice()
	s.lastGridPrice = s.currentGridPrice()

	p.publish("savingsTotalCharged", s.TotalCharged())
	p.publish("savingsGridCharged", s.gridCharged)
	p.publish("savingsSelfConsumptionCharged", s.selfConsumptionCharged)
	p.publish("savingsSelfConsumptionPercent", s.SelfConsumptionPercent())
	p.publish("savingsEffectivePrice", s.EffectivePrice())
	p.publish("savingsAmount", s.SavingsAmount())
	p.publish("tariffGrid", s.currentGridPrice())
	p.publish("tariffFeedIn", s.currentFeedInPrice())
}
