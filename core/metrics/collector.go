package metrics

import (
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
)

const (
	// groups
	Forecast  = "forecast"
	Battery   = "battery"
	Grid      = "grid"
	PV        = "pv"
	Home      = "home" // meter and group (virtual measurement)
	Loadpoint = "loadpoint"
	Meter     = "meter" // generic meter (ext/aux)
)

type Collector struct {
	entity  entity
	accu    *Accumulator
	started time.Time
}

func NewCollector(group, name string, opt ...func(*Accumulator)) (*Collector, error) {
	entity, err := createEntity(group, name)
	if err != nil {
		return nil, err
	}

	c := &Collector{
		entity: entity,
		accu:   NewAccumulator(opt...),
	}

	return c, nil
}

func createEntity(group, name string) (entity, error) {
	entity := entity{
		Group: group,
		Name:  name,
	}

	if err := db.Instance.Where(&entity).FirstOrCreate(&entity).Error; err != nil {
		return entity, err
	}

	return entity, nil
}

func (c *Collector) process(fun func()) error {
	now := c.accu.clock.Now()

	fun()

	slotStart := now.Truncate(tariff.SlotDuration)

	switch {
	case c.started.IsZero():
		// keep started un-truncated so a mid-slot start stays distinguishable
		c.started = now

	case slotStart.After(c.started):
		// persist the completed slot only if started is the immediately
		// preceding slot boundary - false for the mid-slot first slot and
		// for a slot reached after a data gap
		if c.started.Equal(slotStart.Add(-tariff.SlotDuration)) {
			if err := c.persist(); err != nil {
				return err
			}
		}

		c.started = slotStart

	default:
		return nil
	}

	c.accu.Energy = 0
	c.accu.ReturnEnergy = 0

	return nil
}

func (c *Collector) persist() error {
	return persist(c.entity, c.started, c.accu.Energy, c.accu.ReturnEnergy)
}

func (c *Collector) EnergyProfile(from time.Time) (*[96]float64, error) {
	return energyProfile(c.entity, from)
}

func (c *Collector) SetEnergyMeterTotal(v float64) error {
	return c.process(func() {
		c.accu.SetEnergyMeterTotal(v)
	})
}

func (c *Collector) SetReturnEnergyMeterTotal(v float64) error {
	return c.process(func() {
		c.accu.SetReturnEnergyMeterTotal(v)
	})
}

// AddEnergy adds energy using meter totals if available, falling back to power integration.
func (c *Collector) AddEnergy(energyTotal, returnEnergyTotal *float64, power float64) error {
	return c.process(func() {
		switch {
		case energyTotal != nil && returnEnergyTotal != nil:
			c.accu.SetEnergyMeterTotal(*energyTotal)
			c.accu.SetReturnEnergyMeterTotal(*returnEnergyTotal)
		case energyTotal != nil:
			// return energy via power integration (before meter updates clock)
			if power < 0 {
				c.accu.AddPower(power)
			}
			c.accu.SetEnergyMeterTotal(*energyTotal)
		case returnEnergyTotal != nil:
			// energy via power integration (before meter updates clock)
			if power >= 0 {
				c.accu.AddPower(power)
			}
			c.accu.SetReturnEnergyMeterTotal(*returnEnergyTotal)
		default:
			c.accu.AddPower(power)
		}
	})
}
