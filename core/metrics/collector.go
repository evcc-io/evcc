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
	Meter     = "meter"    // additional meter (ext, monitoring only)
	Consumer  = "consumer" // consumer meter (consumers list or aux)
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

// createEntity ensures the entity row exists and refreshes its title.
func createEntity(group, name, title string) (entity, error) {
	// keep history when a meter is regrouped "meter" -> "consumer" (aux, ext convert)
	if group == Consumer {
		var prev entity
		if db.Instance.Where(`"group" = ? AND name = ?`, Meter, name).Limit(1).Find(&prev).RowsAffected > 0 {
			db.Instance.Model(&prev).UpdateColumn("group", Consumer)
		}
	}

	e := entity{Group: group, Name: name}

	if err := db.Instance.Where(&e).Attrs(entity{Title: title}).FirstOrCreate(&e).Error; err != nil {
		return e, err
	}

	return e, e.updateTitle(title)
}

// updateTitle refreshes the entity's stored title if it changed
func (e *entity) updateTitle(title string) error {
	if title == "" || e.Title == title {
		return nil
	}

	e.Title = title
	return db.Instance.Model(e).UpdateColumn("title", title).Error
}

// updateIsTemp refreshes the entity's stored is_temp flag if it changed
func (e *entity) updateIsTemp(isTemp bool) error {
	if e.IsTemp == isTemp {
		return nil
	}

	e.IsTemp = isTemp
	return db.Instance.Model(e).UpdateColumn("is_temp", isTemp).Error
}

// UpdateTitle refreshes the collector entity's stored title if it changed.
func (c *Collector) UpdateTitle(title string) error {
	return c.entity.updateTitle(title)
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
	c.accu.SocTemp = nil
	return nil
}

func (c *Collector) persist() error {
	return persist(c.entity, c.started, c.accu.Energy, c.accu.ReturnEnergy, c.accu.SocTemp)
}

// SetSocTemp records the slot-start soc (temperature when isTemp). Call after AddEnergy.
func (c *Collector) SetSocTemp(value float64, isTemp bool) error {
	c.accu.setSocTemp(value)
	return c.entity.updateIsTemp(isTemp)
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

// AddEnergy adds energy using meter totals if available, falling back to power
// integration only for directions without an energy meter. A direction that has
// reported a total before keeps using meter deltas even if a single read fails,
// so a transient failure is recovered via the next delta and not double-counted.
func (c *Collector) AddEnergy(energyTotal, returnEnergyTotal *float64, power float64) error {
	return c.process(func() {
		// a direction that ever reported a total is metered, so a nil read is a
		// transient failure rather than a power-only meter
		hasEnergyMeter := energyTotal != nil || c.accu.energyMeter != nil
		hasReturnMeter := returnEnergyTotal != nil || c.accu.returnEnergyMeter != nil

		// integrate power for the unmetered direction first, since applying a
		// meter total advances the accumulator clock
		if power >= 0 {
			if !hasEnergyMeter {
				c.accu.AddPower(power)
			}
		} else if !hasReturnMeter {
			c.accu.AddPower(power)
		}

		if energyTotal != nil {
			c.accu.SetEnergyMeterTotal(*energyTotal)
		}
		if returnEnergyTotal != nil {
			c.accu.SetReturnEnergyMeterTotal(*returnEnergyTotal)
		}
	})
}
