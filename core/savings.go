package core

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/tariff"
)

const (
	DefaultGridPrice   = 0.30
	DefaultFeedInPrice = 0.08
)

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
	gridSavedCost          float64   // Running total of saved cost from self consumption (e.g. EUR)
	selfConsumptionCharged float64   // Self-produced energy charged since startup (kWh)
	selfConsumptionCost    float64   // Running total of charged self-produced energy cost (e.g. EUR)
	hasPublished           bool      // Has initial publish happened?
}

func NewSavings(tariffs tariff.Tariffs) *Savings {
	clock := clock.New()
	savings := &Savings{
		clock:   clock,
		tariffs: tariffs,
		started: clock.Now(),
		updated: clock.Now(),
	}

	savings.load()

	return savings
}

func (s *Savings) load() {
	s.started, _ = settings.Time("savings.started")
	s.gridCharged, _ = settings.Float("savings.gridCharged")
	s.gridCost, _ = settings.Float("savings.gridCost")
	s.gridSavedCost, _ = settings.Float("savings.gridSavedCost")
	s.selfConsumptionCharged, _ = settings.Float("savings.selfConsumptionCharged")
	s.selfConsumptionCost, _ = settings.Float("savings.selfConsumptionCost")
}

func (s *Savings) save() {
	settings.SetTime("savings.started", s.started)
	settings.SetFloat("savings.gridCharged", s.gridCharged)
	settings.SetFloat("savings.gridCost", s.gridCost)
	settings.SetFloat("savings.gridSavedCost", s.gridSavedCost)
	settings.SetFloat("savings.selfConsumptionCharged", s.selfConsumptionCharged)
	settings.SetFloat("savings.selfConsumptionCost", s.selfConsumptionCost)
}

func (s *Savings) Since() time.Time {
	return s.started
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
	return s.gridSavedCost
}

func (s *Savings) currentGridPrice() float64 {
	price, err := s.tariffs.CurrentGridPrice()
	if err != nil {
		price = DefaultGridPrice
	}
	return price
}

func (s *Savings) currentFeedInPrice() float64 {
	price, err := s.tariffs.CurrentFeedInPrice()
	if err != nil {
		price = DefaultFeedInPrice
	}
	return price
}

// Update savings calculation and return grid/green energy added since last update
func (s *Savings) Update(p publisher, greenShare, chargePower float64) float64 {
	defer func() { s.updated = s.clock.Now() }()

	// no charging, no need to update
	if chargePower == 0 && s.hasPublished {
		return 0
	}

	// assume charge power as constant over the duration -> rough kWh estimate
	deltaCharged := s.clock.Since(s.updated).Hours() * chargePower / 1e3

	deltaSelf := deltaCharged * greenShare
	deltaGrid := deltaCharged - deltaSelf

	gridPrice := s.currentGridPrice()
	feedInPrice := s.currentFeedInPrice()

	s.gridCharged += deltaGrid
	s.gridCost += deltaGrid * gridPrice
	s.gridSavedCost += deltaSelf * (gridPrice - feedInPrice)
	s.selfConsumptionCharged += deltaSelf
	s.selfConsumptionCost += deltaSelf * feedInPrice

	p.publish("savingsTotalCharged", s.TotalCharged())
	p.publish("savingsGridCharged", s.gridCharged)
	p.publish("savingsSelfConsumptionCharged", s.selfConsumptionCharged)
	p.publish("savingsSelfConsumptionPercent", s.SelfConsumptionPercent())
	p.publish("savingsEffectivePrice", s.EffectivePrice())
	p.publish("savingsAmount", s.SavingsAmount())

	s.hasPublished = true

	s.save()

	return deltaCharged
}
