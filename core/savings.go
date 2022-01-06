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
	chargedTotal           float64   // Energy charged since startup (kWh)
	chargedSelfConsumption float64   // Self-produced energy charged since startup (kWh)
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
	if s.chargedTotal == 0 {
		return 0
	}
	return s.chargedSelfConsumption / s.chargedTotal * 100
}

func (s *Savings) ChargedTotal() float64 {
	return s.chargedTotal
}

func (s *Savings) ChargedSelfConsumption() float64 {
	return s.chargedSelfConsumption
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

func (s *Savings) Update(p publisher, gridPower, pvPower, batteryPower, chargePower float64) {
	// assume charge power as constant over the duration -> rough kWh estimate
	addedEnergy := s.clock.Since(s.updated).Hours() * chargePower / 1e3
	share := s.shareOfSelfProducedEnergy(gridPower, pvPower, batteryPower)

	s.chargedTotal += addedEnergy
	s.chargedSelfConsumption += addedEnergy * share
	s.updated = s.clock.Now()

	s.log.DEBUG.Printf("%.1fkWh charged since %s", s.chargedTotal, time.Since(s.started).Round(time.Second))
	s.log.DEBUG.Printf("%.1fkWh own energy (%.1f%%)", s.chargedSelfConsumption, s.SelfPercentage())

	p.publish("savingsChargedTotal", s.ChargedTotal())
	p.publish("savingsChargedSelfConsumption", s.ChargedSelfConsumption())
	p.publish("savingsSelfPercentage", s.SelfPercentage())

	if s.tariffs.Grid != nil {
		if gridPrice, err := s.tariffs.Grid.CurrentPrice(); err == nil {
			p.publish("tariffGrid", gridPrice)
		}
	}
	if s.tariffs.FeedIn != nil {
		if feedInPrice, err := s.tariffs.FeedIn.CurrentPrice(); err == nil {
			p.publish("tariffFeedIn", feedInPrice)
		}
	}
}
