//go:build linux

package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/stianeikeland/go-rpio/v4"
)

func init() {
	registry.AddCtx("gpio", NewGpioPluginFromConfig)
}

type gpio struct {
	mu  sync.Mutex
	typ GpioType
	pin rpio.Pin
}

// NewGpioPluginFromConfig creates a GPIO provider
func NewGpioPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Function GpioType
		Pin      int
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p := &gpio{
		typ: cc.Function,
		pin: rpio.Pin(cc.Pin),
	}

	// initialize GPIO and set pins to input
	if err := rpio.Open(); err != nil {
		return nil, fmt.Errorf("failed to open GPIO: %w", err)
	}
	defer rpio.Close()

	switch cc.Function {
	case GpioTypeRead:
		p.pin.Input()
	case GpioTypeWrite:
		p.pin.Output()
	default:
		return nil, fmt.Errorf("invalid type: %s", cc.Function)
	}

	return p, nil
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

		if err := rpio.Open(); err != nil {
			return false, fmt.Errorf("failed to open GPIO: %w", err)
		}
		defer rpio.Close()

		return p.pin.Read() != rpio.Low, nil
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

		if err := rpio.Open(); err != nil {
			return fmt.Errorf("failed to open GPIO: %w", err)
		}
		defer rpio.Close()

		val := map[bool]rpio.State{false: rpio.Low, true: rpio.High}[b]
		p.pin.Write(val)

		return nil
	}, nil
}
