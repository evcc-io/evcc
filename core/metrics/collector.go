package metrics

import (
	"time"

	"github.com/evcc-io/evcc/server/db"
)

const (
	SlotDuration = 15 * time.Minute

	// groups
	Virtual = "virtual"
	Grid    = "grid"
	PV      = "pv"

	// meters
	Home = "home" // virtual home meter
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

	return &Collector{
		entity: entity,
		accu:   NewAccumulator(opt...),
	}, nil
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

	if c.accu.updated.IsZero() {
		c.started = now
	}

	fun()

	if slotStart := now.Truncate(SlotDuration); slotStart.After(c.started) {
		// full slot completed
		if slotStart.Sub(c.started) == SlotDuration {
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
	return persist(c.entity, c.started, c.accu.PosEnergy(), c.accu.NegEnergy())
}

func (c *Collector) Profile(from time.Time) (*[96]float64, error) {
	return profile(c.entity, from)
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

func (c *Collector) AddPower(v float64) error {
	return c.process(func() {
		c.accu.AddPower(v)
	})
}
