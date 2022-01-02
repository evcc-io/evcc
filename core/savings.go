package core

import (
	"math"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

// Site is the main configuration container. A site can host multiple loadpoints.
type Savings struct {
	log *util.Logger

	started                time.Time // Boot time
	updated                time.Time // Time of last charged value update
	chargedTotal           float64   // Energy charged since startup (kWh)
	chargedSelfConsumption float64   // Self-produced energy charged since startup (kWh)
	Clock                  clock.Clock
}

func NewSavings() *Savings {
	clock := clock.New()
	savings := &Savings{
		log:     util.NewLogger("savings"),
		started: clock.Now(),
		updated: clock.Now(),
		Clock:   clock,
	}

	return savings
}

func (s *Savings) Since() time.Duration {
	return time.Since(s.started)
}

func (s *Savings) SelfPercentage() float64 {
	if s.chargedTotal == 0 {
		return 0
	}
	return 100 / s.chargedTotal * s.chargedSelfConsumption
}

func (s *Savings) ChargedTotal() float64 {
	return s.chargedTotal
}

func (s *Savings) ChargedSelfConsumption() float64 {
	return s.chargedSelfConsumption
}

func (s *Savings) shareOfSelfProducedEnergy(gridPower float64, pvPower float64, batteryPower float64) float64 {
	batteryDischarge := math.Max(0, batteryPower)
	batteryCharge := math.Min(0, batteryPower) * -1
	pvConsumption := math.Min(pvPower, pvPower+gridPower-batteryCharge)

	gridImport := math.Max(0, gridPower)
	selfConsumption := math.Max(0, batteryDischarge+pvConsumption+batteryCharge)

	selfPercentage := 100 / (gridImport + selfConsumption) * selfConsumption

	if math.IsNaN(selfPercentage) {
		return 0
	}

	return selfPercentage
}

func (s *Savings) Update(gridPower float64, pvPower float64, batteryPower float64, chargePower float64) {
	now := s.Clock.Now()

	selfPercentage := s.shareOfSelfProducedEnergy(gridPower, pvPower, batteryPower)

	updateDuration := now.Sub(s.updated)

	// assuming the charge power was constant over the duration -> rough estimate
	addedEnergy := updateDuration.Hours() * chargePower / 1000

	s.chargedTotal += addedEnergy
	s.chargedSelfConsumption += addedEnergy * (selfPercentage / 100)
	s.updated = now

	s.log.DEBUG.Printf("%.1fkWh charged since %s", s.chargedTotal, time.Since(s.started).Round(time.Second))
	s.log.DEBUG.Printf("%.1fkWh own energy (%.1f%%)", s.chargedSelfConsumption, s.SelfPercentage())
}
