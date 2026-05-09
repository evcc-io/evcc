package metrics

import (
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
)

const (
	// groups
	Battery   = "battery"
	Grid      = "grid"
	PV        = "pv"
	Home      = "home" // meter and group (virtual measurement)
	Loadpoint = "loadpoint"
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

	if slotStart := now.Truncate(tariff.SlotDuration); slotStart.After(c.started) {
		// skip incomplete first slot
		if !c.started.IsZero() {
			if err := c.persist(); err != nil {
				return err
			}
		}

		c.started = slotStart
		c.accu.Import = 0
		c.accu.Export = 0
	}

	return nil
}

func (c *Collector) persist() error {
	return persist(c.entity, c.started, c.accu.Imported(), c.accu.Exported())
}

func (c *Collector) ImportProfile(from time.Time) (*[96]float64, error) {
	return importProfile(c.entity, from)
}

func (c *Collector) AddImportEnergy(v float64) error {
	return c.process(func() {
		c.accu.AddImportEnergy(v)
	})
}

func (c *Collector) AddExportEnergy(v float64) error {
	return c.process(func() {
		c.accu.AddExportEnergy(v)
	})
}

func (c *Collector) SetImportMeterTotal(v float64) error {
	return c.process(func() {
		c.accu.SetImportMeterTotal(v)
	})
}

func (c *Collector) SetExportMeterTotal(v float64) error {
	return c.process(func() {
		c.accu.SetExportMeterTotal(v)
	})
}

// AddEnergy adds energy using meter totals if available, falling back to power integration.
func (c *Collector) AddEnergy(importTotal, exportTotal *float64, power float64) error {
	return c.process(func() {
		switch {
		case importTotal != nil && exportTotal != nil:
			c.accu.SetImportMeterTotal(*importTotal)
			c.accu.SetExportMeterTotal(*exportTotal)
		case importTotal != nil:
			// export via power integration (before meter updates clock)
			if power < 0 {
				c.accu.AddPower(power)
			}
			c.accu.SetImportMeterTotal(*importTotal)
		case exportTotal != nil:
			// import via power integration (before meter updates clock)
			if power >= 0 {
				c.accu.AddPower(power)
			}
			c.accu.SetExportMeterTotal(*exportTotal)
		default:
			c.accu.AddPower(power)
		}
	})
}
