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
	pin rpio.Pin
}

// NewGpioPluginFromConfig creates a GPIO provider
func NewGpioPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Pin int
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p := &gpio{
		pin: rpio.Pin(cc.Pin),
	}

	// initialize GPIO and set pins to input
	if err := rpio.Open(); err != nil {
		return nil, fmt.Errorf("failed to open GPIO: %w", err)
	}
	defer rpio.Close()

	p.pin.Input()

	return p, nil
}

var _ BoolGetter = (*gpio)(nil)

// BoolGetter returns GPIO pin active
func (p *gpio) BoolGetter() (func() (bool, error), error) {
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
