//go:build linux

package plugin

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/warthog618/go-gpiocdev"
)

func init() {
	registry.AddCtx("gpio", NewGpioPluginFromConfig)
}

var gpioBiasOptions = map[GpioBias]gpiocdev.LineBias{
	GpioBiasAsIs:     gpiocdev.WithBiasAsIs,
	GpioBiasDisabled: gpiocdev.WithBiasDisabled,
	GpioBiasPullUp:   gpiocdev.WithPullUp,
	GpioBiasPullDown: gpiocdev.WithPullDown,
}

// sharedLine wraps a gpiocdev.Line shared by all gpio plugin instances requesting the
// same chip+pin, since the kernel only grants one exclusive line request per GPIO offset.
type sharedLine struct {
	mu       sync.Mutex
	line     *gpiocdev.Line
	isOutput bool
	applied  *GpioBias               // explicit bias currently set on the line
	claims   map[*gpioClaim]struct{} // live instances using the line
}

// gpioClaim is held by a plugin instance until its context is done
type gpioClaim struct {
	bias *GpioBias
}

var (
	linesMu sync.Mutex
	lines   = make(map[string]*sharedLine)
)

// claimedBias returns the explicit bias of any live claim
func (sl *sharedLine) claimedBias() *GpioBias {
	for c := range sl.claims {
		if c.bias != nil {
			return c.bias
		}
	}
	return nil
}

// acquireLine returns the shared line for chip+pin, requesting it if not yet open.
// An input line is reconfigured to output on demand, since output values can still be read back.
// The bias is a property of the physical pin and must not conflict between live instances.
// The returned release func drops the claim and closes the line once unused.
func acquireLine(chip string, pin int, output bool, initial int, bias *GpioBias) (*sharedLine, func(), error) {
	linesMu.Lock()
	defer linesMu.Unlock()

	key := chip + ":" + strconv.Itoa(pin)
	claim := &gpioClaim{bias: bias}

	release := func(sl *sharedLine) func() {
		return func() {
			linesMu.Lock()
			defer linesMu.Unlock()

			sl.mu.Lock()
			defer sl.mu.Unlock()

			delete(sl.claims, claim)
			if len(sl.claims) == 0 {
				_ = sl.line.Close()
				delete(lines, key)
			}
		}
	}

	if sl, ok := lines[key]; ok {
		sl.mu.Lock()
		defer sl.mu.Unlock()

		var opts []gpiocdev.LineConfigOption

		if bias != nil {
			if cb := sl.claimedBias(); cb != nil && *cb != *bias {
				return nil, nil, fmt.Errorf("conflicting GPIO bias: requested %s, pin already uses %s", bias, cb)
			}
			if sl.applied == nil || *sl.applied != *bias {
				opts = append(opts, gpioBiasOptions[*bias])
			}
		}

		if output && !sl.isOutput {
			opts = append(opts, gpiocdev.AsOutput(initial))
		}

		if len(opts) > 0 {
			if err := sl.line.Reconfigure(opts...); err != nil {
				return nil, nil, fmt.Errorf("failed to reconfigure GPIO: %w", err)
			}
			sl.isOutput = sl.isOutput || output
			if bias != nil {
				sl.applied = bias
			}
		}

		sl.claims[claim] = struct{}{}

		return sl, release(sl), nil
	}

	var opts []gpiocdev.LineReqOption

	if output {
		opts = append(opts, gpiocdev.AsOutput(initial))
	} else {
		opts = append(opts, gpiocdev.AsInput)
	}

	switch {
	case bias != nil:
		opts = append(opts, gpioBiasOptions[*bias])
	case !output:
		opts = append(opts, gpiocdev.WithPullUp)
	}

	line, err := gpiocdev.RequestLine(chip, pin, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open GPIO: %w", err)
	}

	sl := &sharedLine{
		line:     line,
		isOutput: output,
		applied:  bias,
		claims:   map[*gpioClaim]struct{}{claim: {}},
	}
	lines[key] = sl

	return sl, release(sl), nil
}

type gpio struct {
	typ       GpioType
	activeLow bool
	shared    *sharedLine
}

// NewGpioPluginFromConfig creates a GPIO provider
func NewGpioPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Function  GpioType
		Pin       int
		ActiveLow bool
		Bias      *GpioBias
		Chip      string
	}{
		Chip: "gpiochip0",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	switch cc.Function {
	case GpioTypeRead, GpioTypeWrite:
	default:
		return nil, fmt.Errorf("invalid type: %s", cc.Function)
	}

	if cc.Bias != nil && !cc.Bias.IsAGpioBias() {
		return nil, fmt.Errorf("invalid bias: %s", cc.Bias)
	}

	p := &gpio{
		typ:       cc.Function,
		activeLow: cc.ActiveLow,
	}

	// active low is applied per instance in software, since instances may share a line
	shared, release, err := acquireLine(cc.Chip, cc.Pin, cc.Function == GpioTypeWrite, p.physical(false), cc.Bias)
	if err != nil {
		return nil, err
	}

	p.shared = shared
	context.AfterFunc(ctx, release)

	return p, nil
}

// physical converts a logical state to the physical line value
func (p *gpio) physical(active bool) int {
	if active != p.activeLow {
		return 1
	}
	return 0
}

var _ BoolGetter = (*gpio)(nil)

// BoolGetter returns GPIO pin active
func (p *gpio) BoolGetter() (func() (bool, error), error) {
	if p.typ != GpioTypeRead {
		return nil, fmt.Errorf("invalid gpio type: %s", p.typ)
	}

	return func() (bool, error) {
		p.shared.mu.Lock()
		defer p.shared.mu.Unlock()

		val, err := p.shared.line.Value()
		if err != nil {
			return false, fmt.Errorf("failed to read GPIO: %w", err)
		}

		return (val != 0) != p.activeLow, nil
	}, nil
}

var _ BoolSetter = (*gpio)(nil)

// BoolSetter returns GPIO pin active
func (p *gpio) BoolSetter(_ string) (func(bool) error, error) {
	if p.typ != GpioTypeWrite {
		return nil, fmt.Errorf("invalid gpio type: %s", p.typ)
	}

	return func(b bool) error {
		p.shared.mu.Lock()
		defer p.shared.mu.Unlock()

		if err := p.shared.line.SetValue(p.physical(b)); err != nil {
			return fmt.Errorf("failed to write GPIO: %w", err)
		}

		return nil
	}, nil
}
