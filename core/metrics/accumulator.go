package metrics

import (
	"bytes"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
)

type Accumulator struct {
	clock        clock.Clock
	updated      time.Time
	importMeter  *float64 // kWh
	exportMeter  *float64 // kWh
	Energy       float64  `json:"energy"`       // kWh
	ReturnEnergy float64  `json:"returnEnergy"` // kWh
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
	fmt.Fprintf(b, "Accumulated: %.3fkWh pos, %.3fkWh neg, updated: %v", m.Energy, m.ReturnEnergy, m.updated.Truncate(time.Second))
	if m.importMeter != nil || m.exportMeter != nil {
		fmt.Fprintf(b, " meter: ")
		if m.importMeter != nil {
			fmt.Fprintf(b, " %.3fkWh pos", *m.importMeter)
		}
		if m.exportMeter != nil {
			fmt.Fprintf(b, " %.3fkWh pos", *m.exportMeter)
		}
	}
	return b.String()
}

// Imported returns the accumulated import energy in kWh
func (m *Accumulator) Imported() float64 {
	return m.Energy
}

// Exported returns the accumulated export energy in kWh
func (m *Accumulator) Exported() float64 {
	return m.ReturnEnergy
}

// SetImportMeterTotal adds the difference to the last total meter value in kWh.
// A reading below the last seen total is treated as a transient bad sample and
// ignored. Otherwise a single low/zero read would become the new baseline and
// the next valid read would book the entire lifetime total as a single delta.
func (m *Accumulator) SetImportMeterTotal(v float64) {
	if m.importMeter == nil {
		m.importMeter = new(v)
		m.updated = m.clock.Now()
		return
	}

	if v < *m.importMeter {
		return
	}

	m.Energy += v - *m.importMeter
	m.importMeter = new(v)
	m.updated = m.clock.Now()
}

// SetExportMeterTotal adds the difference to the last total meter value in kWh.
// See SetImportMeterTotal for the rollback handling rationale.
func (m *Accumulator) SetExportMeterTotal(v float64) {
	if m.exportMeter == nil {
		m.exportMeter = new(v)
		m.updated = m.clock.Now()
		return
	}

	if v < *m.exportMeter {
		return
	}

	m.ReturnEnergy += v - *m.exportMeter
	m.exportMeter = new(v)
	m.updated = m.clock.Now()
}

// AddImportEnergy adds the given energy in kWh to the positive meter
func (m *Accumulator) AddImportEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Energy += v
}

// AddExportEnergy adds the given energy in kWh to the negative meter
func (m *Accumulator) AddExportEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.ReturnEnergy += v
}

// AddPower adds the given power in W, calculating the energy based on the time since the last update
func (m *Accumulator) AddPower(v float64) {
	since := v * m.clock.Since(m.updated).Hours() / 1e3
	if v >= 0 {
		m.AddImportEnergy(since)
	} else {
		m.AddExportEnergy(-since)
	}
}
