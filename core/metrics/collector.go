package metrics

import (
	"time"

	"github.com/benbjohnson/clock"
)

const SlotDuration = 15 * time.Minute

type Collector struct {
	accu    Accumulator
	started time.Time
}

func NewCollector(clock clock.Clock) *Collector {
	return &Collector{
		accu: *NewAccumulator(clock),
	}
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
			if err := persist(c.started, c.accu.AccumulatedEnergy()); err != nil {
				return err
			}
		}

		c.started = slotStart
		c.accu.Accumulated = 0
	}

	return nil
}

func (c *Collector) AddEnergy(v float64) error {
	return c.process(func() {
		c.accu.AddEnergy(v)
	})
}

func (c *Collector) AddMeterTotal(v float64) error {
	return c.process(func() {
		c.accu.AddMeterTotal(v)
	})
}

func (c *Collector) AddPower(v float64) error {
	return c.process(func() {
		c.accu.AddPower(v)
	})
}
