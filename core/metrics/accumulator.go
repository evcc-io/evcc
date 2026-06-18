package metrics

import (
	"bytes"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
)

type Accumulator struct {
	clock             clock.Clock
	updated           time.Time
	energyMeter       *float64 // kWh, last accepted total
	returnEnergyMeter *float64 // kWh, last accepted total
	energyReset       *float64 // kWh, pending reset candidate (unconfirmed)
	returnEnergyReset *float64 // kWh, pending reset candidate (unconfirmed)
	Energy            float64  `json:"energy"`       // kWh
	ReturnEnergy      float64  `json:"returnEnergy"` // kWh
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
	fmt.Fprintf(b, "Accumulated: %.3fkWh energy, %.3fkWh return energy, updated: %v", m.Energy, m.ReturnEnergy, m.updated.Truncate(time.Second))
	if m.energyMeter != nil || m.returnEnergyMeter != nil {
		fmt.Fprintf(b, " energy total:")
		if m.energyMeter != nil {
			fmt.Fprintf(b, " %.3fkWh", *m.energyMeter)
		}
		if m.returnEnergyMeter != nil {
			fmt.Fprintf(b, " %.3fkWh return energy", *m.returnEnergyMeter)
		}
	}
	return b.String()
}

// SetEnergyMeterTotal adds the difference to the last total meter value in kWh.
//
// A single reading below the last accepted total (e.g. a transient bad read or
// a device sentinel decoded as 0) must not lower the baseline, otherwise the
// next valid reading would inject a huge bogus delta. A genuine counter reset is
// therefore only accepted once a second, strictly increasing reading confirms
// the counter has resumed counting from a low base.
func (m *Accumulator) SetEnergyMeterTotal(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.energyMeter == nil {
		m.energyMeter = new(v)
		return
	}

	if v >= *m.energyMeter {
		m.Energy += v - *m.energyMeter
		m.energyMeter = new(v)
		m.energyReset = nil
		return
	}

	// counter went backwards: keep the baseline and wait for confirmation
	if m.energyReset != nil && v > *m.energyReset {
		m.Energy += v - *m.energyReset
		m.energyMeter = new(v)
		m.energyReset = nil
		return
	}
	m.energyReset = new(v)
}

// SetReturnEnergyMeterTotal adds the difference to the last total meter value in
// kWh. See [Accumulator.SetEnergyMeterTotal] for the backward-jump handling.
func (m *Accumulator) SetReturnEnergyMeterTotal(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.returnEnergyMeter == nil {
		m.returnEnergyMeter = new(v)
		return
	}

	if v >= *m.returnEnergyMeter {
		m.ReturnEnergy += v - *m.returnEnergyMeter
		m.returnEnergyMeter = new(v)
		m.returnEnergyReset = nil
		return
	}

	// counter went backwards: keep the baseline and wait for confirmation
	if m.returnEnergyReset != nil && v > *m.returnEnergyReset {
		m.ReturnEnergy += v - *m.returnEnergyReset
		m.returnEnergyMeter = new(v)
		m.returnEnergyReset = nil
		return
	}
	m.returnEnergyReset = new(v)
}

// AddEnergy adds the given energy in kWh to the energy total
func (m *Accumulator) AddEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Energy += v
}

// AddReturnEnergy adds the given energy in kWh to the return energy total
func (m *Accumulator) AddReturnEnergy(v float64) {
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
		m.AddEnergy(since)
	} else {
		m.AddReturnEnergy(-since)
	}
}
