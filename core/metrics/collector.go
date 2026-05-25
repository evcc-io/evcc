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

func NewCollector(group, name, title string, opt ...func(*Accumulator)) (*Collector, error) {
	entity, err := createEntity(group, name, title)
	if err != nil {
		return nil, err
	}

	c := &Collector{
		entity: entity,
		accu:   NewAccumulator(opt...),
	}

	return c, nil
}

// createEntity ensures the entity row exists for the given group/name pair and
// refreshes its stored title from the caller. The title is what the UI shows
// for the entity; refreshing on each lazy-create lets the history retain a
// human-readable name even after the device has been renamed or removed.
//
// The where clause matches on (group, name) only - never on title - so an
// existing row whose title differs (including the pre-upgrade empty title) is
// always found and updated in place rather than duplicated.
func createEntity(group, name, title string) (entity, error) {
	e := entity{Group: group, Name: name}

	if err := db.Instance.Where(&e).Attrs(entity{Title: title}).FirstOrCreate(&e).Error; err != nil {
		return e, err
	}

	if title != "" && e.Title != title {
		e.Title = title
		if err := db.Instance.Model(&e).UpdateColumn("title", title).Error; err != nil {
			return e, err
		}
	}

	return e, nil
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
