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

// sharedLine wraps a gpiocdev.Line shared by all gpio plugin instances requesting the
// same chip+pin, since the kernel only grants one exclusive line request per GPIO offset.
type sharedLine struct {
	mu       sync.Mutex
	line     *gpiocdev.Line
	isOutput bool
}

var (
	linesMu sync.Mutex
	lines   = make(map[string]*sharedLine)
)

// acquireLine returns the shared line for chip+pin, requesting it if not yet open.
// An input line is reconfigured to output on demand, since output values can still be read back.
func acquireLine(chip string, pin int, output bool) (*sharedLine, error) {
	linesMu.Lock()
	defer linesMu.Unlock()

	key := chip + ":" + strconv.Itoa(pin)

	if sl, ok := lines[key]; ok {
		if output && !sl.isOutput {
			sl.mu.Lock()
			err := sl.line.Reconfigure(gpiocdev.AsOutput(0))
			if err == nil {
				sl.isOutput = true
			}
			sl.mu.Unlock()
			if err != nil {
				return nil, fmt.Errorf("failed to reconfigure GPIO: %w", err)
			}
		}
		return sl, nil
	}

	var opts []gpiocdev.LineReqOption
	if output {
		opts = append(opts, gpiocdev.AsOutput(0))
	} else {
		opts = append(opts, gpiocdev.AsInput, gpiocdev.WithPullUp)
	}

	line, err := gpiocdev.RequestLine(chip, pin, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPIO: %w", err)
	}

	sl := &sharedLine{line: line, isOutput: output}
	lines[key] = sl

	return sl, nil
}

type gpio struct {
	typ    GpioType
	shared *sharedLine
}

// NewGpioPluginFromConfig creates a GPIO provider
func NewGpioPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Function GpioType
		Pin      int
		Chip     string
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

	shared, err := acquireLine(cc.Chip, cc.Pin, cc.Function == GpioTypeWrite)
	if err != nil {
		return nil, err
	}

	return &gpio{
		typ:    cc.Function,
		shared: shared,
	}, nil
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

		return val != 0, nil
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

		val := 0
		if b {
			val = 1
		}

		if err := p.shared.line.SetValue(val); err != nil {
			return fmt.Errorf("failed to write GPIO: %w", err)
		}

		return nil
	}, nil
}
