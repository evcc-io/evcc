// Ported from shuette42/ecoflow-energy-ha (MIT), see the package documentation.

package ecoflow

import (
	"errors"
	"fmt"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

// Layouts of the charge readings inside a heartbeat.
const (
	LayoutNested   = "nested-field8"
	LayoutTopLevel = "toplevel-28-30"
	LayoutNone     = "no-charge-readings"
)

// State is the wallbox state of heartbeat field 1.
type State uint64

const (
	StateUnplugged        State = 1
	StatePlugged          State = 2
	StateCharging         State = 3
	StatePaused           State = 4 // from the reference implementation, not seen on hardware
	StatePausedByVehicle  State = 5
	StateStoppedByWallbox State = 6
	StateError            State = 9 // from the reference implementation, not seen on hardware
)

// Enabled reports whether the state means charging is released.
func (s State) Enabled() bool {
	switch s {
	case StateCharging, StatePaused, StatePausedByVehicle:
		return true
	default:
		return false
	}
}

// Heartbeat is the 2/33 state report of the PowerPulse.
type Heartbeat struct {
	State State // field 1

	// ChargeCurrentRaw is field 17 in 0.1 A, the session current setpoint. It may differ from
	// the written value by 0.1 A (160 written, 159 reported).
	ChargeCurrentRaw uint64

	// MaxCurrentRaw is field 18 in 0.1 A, the configured maximum.
	MaxCurrentRaw uint64

	PhaseModeRaw uint64 // field 21, the phase mode in effect
	HasPhaseMode bool

	TotalEnergyWh uint64 // field 44, lifetime energy

	PowerW float32
	Volts  []float32
	Amps   []float32
	Layout string
}

// String formats the heartbeat for the trace log.
func (h Heartbeat) String() string {
	return fmt.Sprintf("state=%d power=%.0fW volts=%v amps=%v current(f17)=%d maxCurrent(f18)=%d phaseMode(f21)=%d/%t energy=%dWh layout=%s",
		h.State, h.PowerW, h.Volts, h.Amps, h.ChargeCurrentRaw, h.MaxCurrentRaw, h.PhaseModeRaw, h.HasPhaseMode, h.TotalEnergyWh, h.Layout)
}

// DecodeHeartbeat parses 2/33 pdata. Charge readings in the nested record at
// field 8 win over the top-level fields 28/29/30 of current firmware.
func DecodeHeartbeat(pdata []byte) (Heartbeat, error) {
	fields, err := parseFields(pdata)
	if err != nil {
		return Heartbeat{}, err
	}

	var h Heartbeat
	var haveState, nested bool
	var top Heartbeat // 28/29/30, used only when the nested record is absent
	for _, f := range fields {
		switch {
		case f.Num == 1 && f.Wire == protowire.VarintType:
			if f.V > 255 {
				return Heartbeat{}, fmt.Errorf("implausible heartbeat state %d", f.V)
			}
			h.State, haveState = State(f.V), true
		case f.Num == 8 && f.Wire == protowire.BytesType:
			nested = h.readChargeRecord(f.B)
		case f.Num == 17 && f.Wire == protowire.VarintType:
			h.ChargeCurrentRaw = f.V
		case f.Num == 18 && f.Wire == protowire.VarintType:
			h.MaxCurrentRaw = f.V
		case f.Num == 21 && f.Wire == protowire.VarintType:
			h.PhaseModeRaw, h.HasPhaseMode = f.V, true
		case f.Num == 28 && f.Wire == protowire.Fixed32Type:
			top.PowerW = math.Float32frombits(uint32(f.V))
		case f.Num == 29:
			top.Volts = append(top.Volts, floats(f)...)
		case f.Num == 30:
			top.Amps = append(top.Amps, floats(f)...)
		case f.Num == 44 && f.Wire == protowire.VarintType:
			h.TotalEnergyWh = f.V
		}
	}

	switch {
	case nested:
		h.Layout = LayoutNested
	case top.PowerW != 0 || len(top.Volts) > 0 || len(top.Amps) > 0:
		h.PowerW, h.Volts, h.Amps = top.PowerW, top.Volts, top.Amps
		h.Layout = LayoutTopLevel
	default:
		h.Layout = LayoutNone
	}

	if !haveState {
		return Heartbeat{}, fmt.Errorf("implausible heartbeat: no state")
	}
	for _, v := range append([]float32{h.PowerW}, append(h.Volts, h.Amps...)...) {
		if f := float64(v); math.IsNaN(f) || math.IsInf(f, 0) {
			return Heartbeat{}, fmt.Errorf("implausible heartbeat: charge reading %v", v)
		}
	}
	return h, nil
}

// readChargeRecord decodes the nested charge-readings record: power at 4, the
// three phase voltages at 7..9 and the three phase currents at 10..12.
func (h *Heartbeat) readChargeRecord(b []byte) bool {
	fields, err := parseFields(b)
	if err != nil {
		return false
	}

	volts := make([]float32, 3)
	amps := make([]float32, 3)
	var found bool
	for _, f := range fields {
		vals := floats(f)
		if len(vals) != 1 {
			continue
		}
		switch {
		case f.Num == 4:
			h.PowerW, found = vals[0], true
		case f.Num >= 7 && f.Num <= 9:
			volts[f.Num-7], found = vals[0], true
		case f.Num >= 10 && f.Num <= 12:
			amps[f.Num-10], found = vals[0], true
		}
	}
	if !found {
		return false
	}

	h.Volts, h.Amps = volts, amps
	return true
}

// floats extracts float32 values from either encoding of a repeated float field.
func floats(f field) []float32 {
	switch f.Wire {
	case protowire.Fixed32Type:
		return []float32{math.Float32frombits(uint32(f.V))}
	case protowire.BytesType:
		if len(f.B)%4 != 0 {
			return nil
		}
		out := make([]float32, 0, len(f.B)/4)
		for i := 0; i < len(f.B); i += 4 {
			bits := uint32(f.B[i]) | uint32(f.B[i+1])<<8 | uint32(f.B[i+2])<<16 | uint32(f.B[i+3])<<24
			out = append(out, math.Float32frombits(bits))
		}
		return out
	}
	return nil
}

// Phase modes as carried in 241/44 field 7 and written as 241/102 field 5.
const (
	PhaseAuto  = 0
	PhaseOne   = 1
	PhaseThree = 2
)

// WorkModeCustom is Settings.WorkMode after a 2/81 current write.
const WorkModeCustom = 3

// Settings is a 241/44 configuration report, currents in 0.1 A. Field numbers
// differ from the 241/102 write: max current 3 -> 4, phase mode 5 -> 7.
type Settings struct {
	SwitchBits     uint64 // field 1
	WorkMode       uint64 // field 2: 1 fast, 2 solar, 3 custom, 4 smart
	MaxCurrentRaw  uint64 // field 4
	SolarMinRaw    uint64 // field 6
	PhaseModeRaw   uint64 // field 7: PhaseAuto, PhaseOne or PhaseThree
	UserCurrentRaw uint64 // field 8: the custom current, written by 2/81 (WorkModeCustom)
}

// ErrNoSettings is the shorter 241/44 variant, which carries no settings.
var ErrNoSettings = errors.New("no settings")

var errFieldMissing = errors.New("missing")

// DecodeSettings parses the settings at pdata -> 1 -> 4 -> 8. Any other shape, a
// missing field or an implausible value is rejected, so no variant is half-decoded.
func DecodeSettings(pdata []byte) (Settings, error) {
	b := pdata
	for _, num := range []int{1, 4, 8} {
		fields, err := parseFields(b)
		if err != nil {
			return Settings{}, err
		}
		if b, err = singleBytes(fields, num); err != nil {
			if num == 8 && errors.Is(err, errFieldMissing) {
				return Settings{}, ErrNoSettings
			}
			return Settings{}, err
		}
	}

	params, err := parseFields(b)
	if err != nil {
		return Settings{}, err
	}

	var s Settings
	for _, t := range []struct {
		num      int
		dst      *uint64
		min, max uint64
	}{
		{1, &s.SwitchBits, 0, 0xFFFF},
		{2, &s.WorkMode, 1, 4},
		{4, &s.MaxCurrentRaw, 60, 320},
		{6, &s.SolarMinRaw, 60, 320},
		{7, &s.PhaseModeRaw, 0, 3},
		{8, &s.UserCurrentRaw, 60, 320},
	} {
		v, ok := varintField(params, t.num)
		if !ok {
			return Settings{}, fmt.Errorf("settings field %d missing", t.num)
		}
		if v < t.min || v > t.max {
			return Settings{}, fmt.Errorf("settings field %d = %d out of range %d..%d", t.num, v, t.min, t.max)
		}
		*t.dst = v
	}

	return s, nil
}

// singleBytes returns the payload of num, requiring exactly one occurrence.
// Taking the first of several would silently decode a different variant.
func singleBytes(fields []field, num int) ([]byte, error) {
	var found []byte
	for _, f := range fields {
		if f.Num != num || f.Wire != protowire.BytesType {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("field %d appears more than once", num)
		}
		found = f.B
	}
	if found == nil {
		return nil, fmt.Errorf("field %d %w", num, errFieldMissing)
	}
	return found, nil
}

// varintField returns the first varint value of num.
func varintField(fields []field, num int) (uint64, bool) {
	for _, f := range fields {
		if f.Num == num && f.Wire == protowire.VarintType {
			return f.V, true
		}
	}
	return 0, false
}
