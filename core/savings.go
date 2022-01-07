package core

import (
	"math"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
)

// publisher gives access to the site's publish function
type publisher interface {
	publish(key string, val interface{})
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Savings struct {
	log                    *util.Logger
	clock                  clock.Clock
	tariffs                tariff.Tariffs
	started                time.Time // Boot time
	updated                time.Time // Time of last charged value update
	chargedGrid            float64   // Grid energy charged since startup (kWh)
	chargedSelfConsumption float64   // Self-produced energy charged since startup (kWh)
	costGrid               float64   // Cost of charged grid energy (e.g. EUR)
	costSelfConsumption    float64   // Cost of charged self-produced energy (e.g. EUR)
}

func NewSavings(tariffs tariff.Tariffs) *Savings {
	clock := clock.New()
	savings := &Savings{
		log:     util.NewLogger("savings"),
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

func (s *Savings) SelfPercentage() float64 {
	if s.ChargedTotal() == 0 {
		return 0
	}
	return s.chargedSelfConsumption / s.ChargedTotal() * 100
}

func (s *Savings) ChargedTotal() float64 {
	return s.chargedGrid + s.chargedSelfConsumption
}

func (s *Savings) CostTotal() float64 {
	return s.costGrid + s.costSelfConsumption
}

func (s *Savings) EffectivePrice() float64 {
	if s.ChargedTotal() == 0 {
		return s.currentGridPrice()
	}
	return s.CostTotal() / s.ChargedTotal()
}

func (s *Savings) SavingsAmount() float64 {
	return s.chargedSelfConsumption * (s.currentGridPrice() - s.currentFeedInPrice())
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
	return 0.3
}

func (s *Savings) currentFeedInPrice() float64 {
	if s.tariffs.FeedIn != nil {
		if gridPrice, err := s.tariffs.FeedIn.CurrentPrice(); err == nil {
			return gridPrice
		}
	}
	return 0.08
}

func (s *Savings) Update(p publisher, gridPower, pvPower, batteryPower, chargePower float64) {
	// assume charge power as constant over the duration -> rough kWh estimate
	addedEnergy := s.clock.Since(s.updated).Hours() * chargePower / 1e3
	share := s.shareOfSelfProducedEnergy(gridPower, pvPower, batteryPower)

	addedSelfConsumption := addedEnergy * share
	addedGrid := addedEnergy - addedSelfConsumption

	s.chargedGrid += addedGrid
	s.costGrid += addedGrid * s.currentGridPrice()
	s.chargedSelfConsumption += addedSelfConsumption
	s.costSelfConsumption += addedSelfConsumption * s.currentFeedInPrice()

	s.updated = s.clock.Now()

	s.log.DEBUG.Printf("%.1fkWh charged since %s", s.ChargedTotal(), time.Since(s.started).Round(time.Second))
	s.log.DEBUG.Printf("%.1fkWh own energy (%.1f%%)", s.chargedSelfConsumption, s.SelfPercentage())

	p.publish("savingsChargedTotal", s.ChargedTotal())
	p.publish("savingsChargedGrid", s.chargedGrid)
	p.publish("savingsChargedSelfConsumption", s.chargedSelfConsumption)
	p.publish("savingsSelfPercentage", s.SelfPercentage())
	p.publish("savingsEffectivePrice", s.EffectivePrice())
	p.publish("savingsAmount", s.SavingsAmount())
	p.publish("tariffGrid", s.currentGridPrice())
	p.publish("tariffFeedIn", s.currentFeedInPrice())
}
