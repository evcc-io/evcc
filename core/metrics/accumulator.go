package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
)

type Accumulator struct {
	clock       clock.Clock
	updated     time.Time
	importMeter *float64 // kWh
	exportMeter *float64 // kWh
	Import      float64  `json:"import"` // kWh
	Export      float64  `json:"export"` // kWh
}

type accumulatorJSON struct {
	Import      *float64   `json:"import,omitempty"`
	Export      *float64   `json:"export,omitempty"`
	Updated     *time.Time `json:"updated,omitempty"`
	Accumulated *float64   `json:"accumulated,omitempty"`
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

// Restore sets persisted accumulator values and clears transient meter totals.
func (m *Accumulator) Restore(imp, exp float64, updated time.Time) {
	m.Import = imp
	m.Export = exp
	m.updated = updated
	m.importMeter = nil
	m.exportMeter = nil
}

func (m Accumulator) MarshalJSON() ([]byte, error) {
	payload := accumulatorJSON{
		Import:  &m.Import,
		Export:  &m.Export,
		Updated: &m.updated,
	}

	return json.Marshal(payload)
}

func (m *Accumulator) UnmarshalJSON(data []byte) error {
	var payload accumulatorJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if m.clock == nil {
		m.clock = clock.New()
	}

	var imp float64
	switch {
	case payload.Import != nil:
		imp = *payload.Import
	case payload.Accumulated != nil:
		imp = *payload.Accumulated
	}

	var exp float64
	if payload.Export != nil {
		exp = *payload.Export
	}

	var updated time.Time
	if payload.Updated != nil {
		updated = *payload.Updated
	}

	m.Restore(imp, exp, updated)

	return nil
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

// Imported returns the accumulated import energy in kWh
func (m *Accumulator) Imported() float64 {
	return m.Import
}

// Exported returns the accumulated export energy in kWh
func (m *Accumulator) Exported() float64 {
	return m.Export
}

// SetImportMeterTotal adds the difference to the last total meter value in kWh
func (m *Accumulator) SetImportMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.importMeter = new(v)
	}()

	if m.importMeter == nil {
		return
	}

	if v >= *m.importMeter {
		m.Import += v - *m.importMeter
	}
}

// SetExportMeterTotal adds the difference to the last total meter value in kWh
func (m *Accumulator) SetExportMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.exportMeter = new(v)
	}()

	if m.exportMeter == nil {
		return
	}

	if v >= *m.exportMeter {
		m.Export += v - *m.exportMeter
	}
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
