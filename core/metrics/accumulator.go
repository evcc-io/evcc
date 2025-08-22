package metrics

import (
	"bytes"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/samber/lo"
)

type Accumulator struct {
	clock       clock.Clock
	updated     time.Time
	meter       *float64 // kWh
	Accumulated float64  `json:"accumulated"` // kWh
}

func NewAccumulator(clock clock.Clock) *Accumulator {
	return &Accumulator{clock: clock}
}

func (m *Accumulator) Updated() time.Time {
	return m.updated
}

func (m *Accumulator) String() string {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "Accumulated: %.3fkWh updated: %v", m.Accumulated, m.updated.Truncate(time.Second))
	if m.meter != nil {
		fmt.Fprintf(b, " meter: %.3fkWh", *m.meter)
	}
	return b.String()
}

// AccumulatedEnergy returns the accumulated energy in kWh
func (m *Accumulator) AccumulatedEnergy() float64 {
	return m.Accumulated
}

// AddMeterTotal adds the difference to the last total meter value in kWh
func (m *Accumulator) AddMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.meter = lo.ToPtr(v)
	}()

	if m.meter == nil {
		return
	}

	m.Accumulated += v - *m.meter
}

// AddEnergy adds the given energy in kWh
func (m *Accumulator) AddEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Accumulated += v
}

// AddPower adds the given power in W, calculating the energy based on the time since the last update
func (m *Accumulator) AddPower(v float64) {
	m.AddEnergy(v * m.clock.Since(m.updated).Hours() / 1e3)
}
