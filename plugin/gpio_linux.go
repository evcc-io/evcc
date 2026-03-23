//go:build linux

package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/warthog618/go-gpiocdev"
)

func init() {
	registry.AddCtx("gpio", NewGpioPluginFromConfig)
}

type gpio struct {
	mu   sync.Mutex
	typ  GpioType
	line *gpiocdev.Line
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

	var opts []gpiocdev.LineReqOption
	switch cc.Function {
	case GpioTypeRead:
		opts = append(opts, gpiocdev.AsInput, gpiocdev.WithPullUp)
	case GpioTypeWrite:
		opts = append(opts, gpiocdev.AsOutput(0))
	default:
		return nil, fmt.Errorf("invalid type: %s", cc.Function)
	}

	line, err := gpiocdev.RequestLine(cc.Chip, cc.Pin, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPIO: %w", err)
	}

	return &gpio{
		typ:  cc.Function,
		line: line,
	}, nil
}

var _ BoolGetter = (*gpio)(nil)

// BoolGetter returns GPIO pin active
func (p *gpio) BoolGetter() (func() (bool, error), error) {
	if p.typ != GpioTypeRead {
		return nil, fmt.Errorf("invalid gpio type: %s", p.typ)
	}

	return func() (bool, error) {
		p.mu.Lock()
		defer p.mu.Unlock()

		val, err := p.line.Value()
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
		p.mu.Lock()
		defer p.mu.Unlock()

		val := 0
		if b {
			val = 1
		}

		if err := p.line.SetValue(val); err != nil {
			return fmt.Errorf("failed to write GPIO: %w", err)
		}

		return nil
	}, nil
}
