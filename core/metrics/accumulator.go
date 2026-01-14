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
	importMeter *float64 // kWh
	exportMeter *float64 // kWh
	Import      float64  `json:"import"` // kWh
	Export      float64  `json:"export"` // kWh
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
	fmt.Fprintf(b, "Accumulated: %.3fkWh pos, %.3fkWh neg, updated: %v", m.Import, m.Export, m.updated.Truncate(time.Second))
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

// PosEnergy returns the accumulated energy in kWh
func (m *Accumulator) PosEnergy() float64 {
	return m.Import
}

// NegEnergy returns the accumulated energy in kWh
func (m *Accumulator) NegEnergy() float64 {
	return m.Export
}

// SetImportMeterTotal adds the difference to the last total meter value in kWh
func (m *Accumulator) SetImportMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.importMeter = lo.ToPtr(v)
	}()

	if m.importMeter == nil {
		return
	}

	m.Import += v - *m.importMeter
}

// SetExportMeterTotal adds the difference to the last total meter value in kWh
func (m *Accumulator) SetExportMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.exportMeter = lo.ToPtr(v)
	}()

	if m.exportMeter == nil {
		return
	}

	m.Export += v - *m.exportMeter
}

// AddImportEnergy adds the given energy in kWh to the positive meter
func (m *Accumulator) AddImportEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Import += v
}

// AddExportEnergy adds the given energy in kWh to the negative meter
func (m *Accumulator) AddExportEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Export += v
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
