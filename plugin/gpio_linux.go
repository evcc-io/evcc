//go:build linux

package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stianeikeland/go-rpio/v4"
)

func init() {
	registry.AddCtx("gpio", NewGpioPluginFromConfig)
}

type gpio struct {
	mu     sync.Mutex
	pin    rpio.Pin
	active bool
}

// NewGpioPluginFromConfig creates a GPIO provider
func NewGpioPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Pin int
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// initialize GPIO and set pins to output
	if err := rpio.Open(); err != nil {
		return nil, fmt.Errorf("failed to open GPIO: %w", err)
	}

	p := &gpio{
		pin: rpio.Pin(cc.Pin),
	}

	p.pin.Input()

	go p.run(ctx)

	return &p, nil
}

func (p *gpio) run(ctx context.Context) {
	for tick := time.Tick(time.Second); ; {
		select {
		case <-ctx.Done():
			rpio.Close()

		case <-tick:
			val := p.pin.Read()

			p.mu.Lock()
			p.active = val != rpio.Low
			p.mu.Unlock()
		}
	}
}

var _ BoolGetter = (*gpio)(nil)

// BoolGetter returns GPIO pin active
func (p *gpio) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.active, nil
	}, nil
}
