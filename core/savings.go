package core

import (
	"fmt"
	"math"
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

func load(name string, res any) {
	key := "savings." + name
	store := settings.NewStore(key)
	if err := store.Load(&res); err != nil {
		fmt.Printf("no value for %s in database. first start.\n", key)
	}
}

func save(name string, value any) {
	key := "savings." + name
	store := settings.NewStore(key)
	if err := store.Save(value); err != nil {
		fmt.Printf("unable to save %s settings.\n", key)
	}
}

// Site is the main configuration container. A site can host multiple loadpoints.
type Savings struct {
	clock                          clock.Clock
	tariffs                        tariff.Tariffs
	started                        time.Time    // Boot time
	updated                        time.Time    // Time of last charged value update
	gridCharged                    float64      // Grid energy charged since startup (kWh)
	gridCost                       float64      // Running total of charged grid energy cost (e.g. EUR)
	gridSavedCost                  float64      // Running total of saved cost from self consumption (e.g. EUR)
	selfConsumptionCharged         float64      // Self-produced energy charged since startup (kWh)
	selfConsumptionCost            float64      // Running total of charged self-produced energy cost (e.g. EUR)
	lastGridPrice, lastFeedInPrice float64      // Stores the last published grid price. Needed to detect price changes (Awattar, ..)
	persistTicker                  *time.Ticker // Ticker that defines the interval when savings are written to disk
}

func NewSavings(tariffs tariff.Tariffs) *Savings {
	clock := clock.New()
	savings := &Savings{
		clock:         clock,
		tariffs:       tariffs,
		started:       clock.Now(),
		updated:       clock.Now(),
		persistTicker: time.NewTicker(time.Minute),
	}

	savings.restore()

	return savings
}

func (s *Savings) restore() {
	load("started", &s.started)
	load("gridCharged", &s.gridCharged)
	load("gridCost", &s.gridCost)
	load("gridSavedCost", &s.gridSavedCost)
	load("selfConsumptionCharged", &s.selfConsumptionCharged)
	load("selfConsumptionCost", &s.selfConsumptionCost)

	// fmt.Println("restored savings.*")
}

func (s *Savings) save() {
	save("started", s.started)
	save("gridCharged", s.gridCharged)
	save("gridCost", s.gridCost)
	save("gridSavedCost", s.gridSavedCost)
	save("selfConsumptionCharged", s.selfConsumptionCharged)
	save("selfConsumptionCost", s.selfConsumptionCost)

	quit := make(chan struct{})
	for {
		select {
		case <-s.persistTicker.C:
			if err := settings.Persist(); err != nil {
				fmt.Printf("unable to persist settings.\n")
				return
			}
			// fmt.Println("saved savings.*")
		case <-quit:
			s.persistTicker.Stop()
			return
		}
	}
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

func (s *Savings) updatePrices(p publisher) (float64, float64) {
	gridPrice := s.currentGridPrice()
	if gridPrice != s.lastGridPrice {
		s.lastGridPrice = gridPrice
		p.publish("tariffGrid", gridPrice)
	}

	feedinPrice := s.currentFeedInPrice()
	if feedinPrice != s.lastFeedInPrice {
		s.lastFeedInPrice = feedinPrice
		p.publish("tariffFeedIn", feedinPrice)
	}

	return gridPrice, feedinPrice
}

// Update savings calculation and return grid/green energy added since last update
func (s *Savings) Update(p publisher, gridPower, pvPower, batteryPower, chargePower float64) (float64, float64) {
	gridPrice, feedinPrice := s.updatePrices(p)
	defer func() { s.updated = s.clock.Now() }()

	// no charging, no need to update
	if chargePower == 0 {
		return 0, 0
	}

	// assume charge power as constant over the duration -> rough kWh estimate
	deltaCharged := s.clock.Since(s.updated).Hours() * chargePower / 1e3
	share := s.shareOfSelfProducedEnergy(gridPower, pvPower, batteryPower)

	deltaSelf := deltaCharged * share
	deltaGrid := deltaCharged - deltaSelf

	s.gridCharged += deltaGrid
	s.gridCost += deltaGrid * gridPrice
	s.gridSavedCost += deltaSelf * (gridPrice - feedinPrice)
	s.selfConsumptionCharged += deltaSelf
	s.selfConsumptionCost += deltaSelf * feedinPrice

	p.publish("savingsTotalCharged", s.TotalCharged())
	p.publish("savingsGridCharged", s.gridCharged)
	p.publish("savingsSelfConsumptionCharged", s.selfConsumptionCharged)
	p.publish("savingsSelfConsumptionPercent", s.SelfConsumptionPercent())
	p.publish("savingsEffectivePrice", s.EffectivePrice())
	p.publish("savingsAmount", s.SavingsAmount())

	s.save()

	return deltaCharged, deltaSelf
}
