package metrics

import (
	"bytes"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/samber/lo"
)

type Accumulator struct {
	clock              clock.Clock
	updated            time.Time
	posMeter, negMeter *float64 // kWh
	Pos                float64  `json:"pos"` // kWh
	Neg                float64  `json:"neg"` // kWh
}

func WithClock(clock clock.Clock) func(*Accumulator) {
	return func(m *Accumulator) {
		m.clock = clock
	}
}

func NewAccumulator(opt ...func(*Accumulator)) *Accumulator {
	m := &Accumulator{clock: clock.New()}
	for _, o := range opt {
		o(m)
	}
	return m
}

func (m *Accumulator) Updated() time.Time {
	return m.updated
}

func (m *Accumulator) String() string {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "Accumulated: %.3fkWh pos, %.3fkWh neg, updated: %v", m.Pos, m.Neg, m.updated.Truncate(time.Second))
	if m.posMeter != nil || m.negMeter != nil {
		fmt.Fprintf(b, " meter: ")
		if m.posMeter != nil {
			fmt.Fprintf(b, " %.3fkWh pos", *m.posMeter)
		}
		if m.negMeter != nil {
			fmt.Fprintf(b, " %.3fkWh pos", *m.negMeter)
		}
	}
	return b.String()
}

// PosEnergy returns the accumulated energy in kWh
func (m *Accumulator) PosEnergy() float64 {
	return m.Pos
}

// NegEnergy returns the accumulated energy in kWh
func (m *Accumulator) NegEnergy() float64 {
	return m.Neg
}

// SetImportMeterTotal adds the difference to the last total meter value in kWh
func (m *Accumulator) SetImportMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.posMeter = lo.ToPtr(v)
	}()

	if m.posMeter == nil {
		return
	}

	m.Pos += v - *m.posMeter
}

// SetExportMeterTotal adds the difference to the last total meter value in kWh
func (m *Accumulator) SetExportMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.negMeter = lo.ToPtr(v)
	}()

	if m.negMeter == nil {
		return
	}

	m.Neg += v - *m.negMeter
}

// AddImportEnergy adds the given energy in kWh to the positive meter
func (m *Accumulator) AddImportEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Pos += v
}

// AddExportEnergy adds the given energy in kWh to the negative meter
func (m *Accumulator) AddExportEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Neg += v
}

// AddPower adds the given power in W, calculating the energy based on the time since the last update
func (m *Accumulator) AddPower(v float64) {
	if v >= 0 {
		m.AddImportEnergy(v * m.clock.Since(m.updated).Hours() / 1e3)
	} else {
		m.AddExportEnergy(-v * m.clock.Since(m.updated).Hours() / 1e3)
	}
}
